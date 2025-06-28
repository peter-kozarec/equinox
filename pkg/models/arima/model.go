package arima

import (
	"errors"
	"github.com/peter-kozarec/equinox/pkg/utility/circular"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"math"
)

var (
	ErrModelNotEstimated    = errors.New("model not estimated - insufficient data or estimation failed")
	ErrInsufficientData     = errors.New("insufficient data points for reliable estimation")
	ErrNumericalInstability = errors.New("numerical instability detected during estimation")
	ErrInvalidParameters    = errors.New("invalid model parameters")
	ErrNonStationarity      = errors.New("series appears non-stationary after differencing")
	ErrConvergenceFailed    = errors.New("parameter estimation failed to converge")
)

const (
	MinDataPoints         = 50   // Minimum data points for reliable estimation
	MaxIterations         = 1000 // Maximum iterations for optimization
	ConvergenceTolerance  = 1e-8 // Convergence tolerance for parameter estimation
	StationarityThreshold = 0.99 // Threshold for stationarity check
)

type EstimationMethod int

const (
	MaximumLikelihood EstimationMethod = iota
	ConditionalLeastSquares
	UnconditionalLeastSquares
	YuleWalker
)

type ModelDiagnostics struct {
	LogLikelihood   fixed.Point
	AIC             fixed.Point
	BIC             fixed.Point
	AICC            fixed.Point
	RMSE            fixed.Point
	MAE             fixed.Point
	MAPE            fixed.Point
	LjungBoxPValue  fixed.Point
	JarqueBeraTest  fixed.Point
	IsStationary    bool
	ConvergenceCode int
	Iterations      int
}

type ForecastResult struct {
	PointForecast      fixed.Point
	StandardError      fixed.Point
	ConfidenceInterval struct {
		Lower95 fixed.Point
		Upper95 fixed.Point
		Lower80 fixed.Point
		Upper80 fixed.Point
	}
	PredictionInterval struct {
		Lower95 fixed.Point
		Upper95 fixed.Point
		Lower80 fixed.Point
		Upper80 fixed.Point
	}
}

type Model struct {
	p, d, q         uint
	includeConstant bool
	seasonal        bool
	seasonalPeriod  uint

	ptCounter       uint
	winSize         uint
	minObservations uint

	rawData  *circular.PointBuffer
	diffData *circular.PointBuffer

	arParams []fixed.Point
	maParams []fixed.Point
	constant fixed.Point
	variance fixed.Point

	residuals             *circular.PointBuffer
	standardizedResiduals *circular.PointBuffer

	estimated   bool
	method      EstimationMethod
	diagnostics ModelDiagnostics

	forecastCache []fixed.Point
}

type forecastState struct {
	diffSeries      []fixed.Point
	residuals       []fixed.Point
	mean            fixed.Point
	variance        fixed.Point
	rawValues       []fixed.Point
	forecastedDiffs []fixed.Point
}

type ModelOption func(*Model)

func NewModel(p, d, q, winSize uint, options ...ModelOption) (*Model, error) {
	if p == 0 && q == 0 {
		return nil, ErrInvalidParameters
	}

	if winSize < MinDataPoints {
		winSize = MinDataPoints
	}

	minObs := max(p+d+q+10, MinDataPoints)

	m := &Model{
		p:                     p,
		d:                     d,
		q:                     q,
		includeConstant:       true,
		seasonal:              false,
		ptCounter:             0,
		winSize:               winSize,
		minObservations:       minObs,
		rawData:               circular.NewPointBuffer(winSize),
		diffData:              circular.NewPointBuffer(winSize - d),
		residuals:             circular.NewPointBuffer(winSize),
		standardizedResiduals: circular.NewPointBuffer(winSize),
		arParams:              make([]fixed.Point, p),
		maParams:              make([]fixed.Point, q),
		constant:              fixed.Zero,
		variance:              fixed.Zero,
		estimated:             false,
		method:                MaximumLikelihood,
		forecastCache:         make([]fixed.Point, 0),
	}

	// Initialize parameters to zero
	for i := uint(0); i < p; i++ {
		m.arParams[i] = fixed.Zero
	}
	for i := uint(0); i < q; i++ {
		m.maParams[i] = fixed.Zero
	}

	// Apply options
	for _, option := range options {
		option(m)
	}

	return m, nil
}

// Model Options

func WithEstimationMethod(method EstimationMethod) ModelOption {
	return func(m *Model) {
		m.method = method
	}
}

func WithConstant(include bool) ModelOption {
	return func(m *Model) {
		m.includeConstant = include
	}
}

func WithSeasonal(period uint) ModelOption {
	return func(m *Model) {
		if period > 1 {
			m.seasonal = true
			m.seasonalPeriod = period
		}
	}
}

// Main Model Methods

// AddPoint adds a new observation to the model
func (m *Model) AddPoint(p fixed.Point) error {
	m.rawData.PushUpdate(p)
	m.ptCounter++

	// Perform differencing
	if m.rawData.B.Size() > m.d {
		diff := m.differenceLatest()
		m.diffData.PushUpdate(diff)
	}

	// Check if we should re-estimate
	if m.shouldReestimate() {
		return m.estimate()
	}

	return nil
}

// Forecast generates h-step ahead forecasts
func (m *Model) Forecast(steps uint) ([]ForecastResult, error) {
	if !m.estimated {
		return nil, ErrModelNotEstimated
	}

	results := make([]ForecastResult, steps)
	m.forecastCache = make([]fixed.Point, 0, steps)

	// Initialize forecast state
	state := m.initializeForecastState()

	for step := uint(0); step < steps; step++ {
		result, err := m.forecastOneStep(state, step)
		if err != nil {
			return nil, err
		}
		results[step] = result
		m.forecastCache = append(m.forecastCache, result.PointForecast)

		// Update forecast state for next step
		m.updateForecastState(state, result.PointForecast, step)
	}

	return results, nil
}

// Public Getter Methods

// GetDiagnostics returns model diagnostics
func (m *Model) GetDiagnostics() ModelDiagnostics {
	return m.diagnostics
}

// GetParameters returns the estimated AR parameters, MA parameters, and constant
func (m *Model) GetParameters() ([]fixed.Point, []fixed.Point, fixed.Point) {
	arCopy := make([]fixed.Point, len(m.arParams))
	copy(arCopy, m.arParams)

	maCopy := make([]fixed.Point, len(m.maParams))
	copy(maCopy, m.maParams)

	return arCopy, maCopy, m.constant
}

// GetResiduals returns the model residuals in oldest-to-newest order
func (m *Model) GetResiduals() []fixed.Point {
	if m.residuals.B.Size() == 0 {
		return []fixed.Point{}
	}

	data := m.residuals.B.Data()
	result := make([]fixed.Point, len(data))

	// Reverse to get oldest to newest order
	for i := 0; i < len(data); i++ {
		result[i] = data[len(data)-1-i]
	}

	return result
}

// GetStandardizedResiduals returns standardized residuals in oldest-to-newest order
func (m *Model) GetStandardizedResiduals() []fixed.Point {
	if m.standardizedResiduals.B.Size() == 0 {
		return []fixed.Point{}
	}

	data := m.standardizedResiduals.B.Data()
	result := make([]fixed.Point, len(data))

	// Reverse to get oldest to newest order
	for i := 0; i < len(data); i++ {
		result[i] = data[len(data)-1-i]
	}

	return result
}

// IsEstimated returns whether the model has been estimated
func (m *Model) IsEstimated() bool {
	return m.estimated
}

// GetVariance returns the estimated error variance
func (m *Model) GetVariance() fixed.Point {
	return m.variance
}

// GetFittedValues returns fitted values from the model
func (m *Model) GetFittedValues() []fixed.Point {
	series := m.getDiffSeriesInOrder()
	startIdx := max(m.p, m.q)

	if uint(len(series)) <= startIdx {
		return []fixed.Point{}
	}

	fitted := make([]fixed.Point, len(series)-int(startIdx))
	residuals := m.calculateCurrentResiduals()

	for i := 0; i < len(fitted) && i < len(residuals); i++ {
		actual := series[int(startIdx)+i]
		fitted[i] = actual.Sub(residuals[i])
	}

	return fitted
}

// ValidateModel performs validation checks on the estimated model
func (m *Model) ValidateModel() error {
	if !m.estimated {
		return ErrModelNotEstimated
	}

	// Check parameter constraints
	if err := m.checkParameterValidity(); err != nil {
		return err
	}

	// Check residual properties
	if err := m.checkResidualProperties(); err != nil {
		return err
	}

	return nil
}

// Reset clears all model state
func (m *Model) Reset() {
	m.rawData.Clear()
	m.diffData.Clear()
	m.residuals.Clear()
	m.standardizedResiduals.Clear()

	// Reset parameters
	for i := range m.arParams {
		m.arParams[i] = fixed.Zero
	}
	for i := range m.maParams {
		m.maParams[i] = fixed.Zero
	}

	m.constant = fixed.Zero
	m.variance = fixed.Zero
	m.estimated = false
	m.ptCounter = 0
	m.forecastCache = make([]fixed.Point, 0)
}

// Private Methods - Differencing

func (m *Model) differenceLatest() fixed.Point {
	if m.rawData.B.Size() <= m.d {
		return fixed.Zero
	}

	// Apply differencing d times using binomial coefficients
	coeffs := m.getDifferencingCoefficients(m.d)
	result := fixed.Zero

	for i, coeff := range coeffs {
		if uint(i) < m.rawData.B.Size() {
			// Get(0) is most recent, Get(1) is one lag back, etc.
			result = result.Add(coeff.Mul(m.rawData.B.Get(uint(i))))
		}
	}

	return result
}

func (m *Model) getDifferencingCoefficients(d uint) []fixed.Point {
	// These are alternating binomial coefficients: (1, -d, d(d-1)/2, ...)
	coeffs := make([]fixed.Point, d+1)
	coeffs[0] = fixed.One

	for k := uint(1); k <= d; k++ {
		coeff := binomialCoefficient(d, k)
		if k%2 == 1 {
			coeff = coeff.Mul(fixed.NegOne)
		}
		coeffs[k] = coeff
	}

	return coeffs
}

func (m *Model) undifferenceWithState(diffValue fixed.Point, state *forecastState, step uint) fixed.Point {
	if m.d == 0 {
		return diffValue
	}

	// Build the complete series including historical and forecasted values
	fullSeries := make([]fixed.Point, 0, len(state.rawValues)+int(step)+1)
	fullSeries = append(fullSeries, state.rawValues...)

	// Add previously forecasted values
	for i := 0; i < int(step) && i < len(m.forecastCache); i++ {
		fullSeries = append(fullSeries, m.forecastCache[i])
	}

	// Apply inverse differencing
	result := diffValue

	if m.d == 1 {
		// Simple case: just add the previous value
		if len(fullSeries) > 0 {
			result = result.Add(fullSeries[len(fullSeries)-1])
		}
	} else {
		// For higher order differencing, sum the appropriate lagged values
		for i := uint(0); i < m.d; i++ {
			idx := len(fullSeries) - 1 - int(i)
			if idx >= 0 && idx < len(fullSeries) {
				result = result.Add(fullSeries[idx])
			}
		}
	}

	return result
}

// Private Methods - Estimation

func (m *Model) shouldReestimate() bool {
	// Re-estimate when we have enough data and either:
	// 1. Model hasn't been estimated yet
	// 2. We've collected a full window of new data
	return m.diffData.B.Size() >= m.minObservations &&
		(!m.estimated || m.ptCounter >= m.winSize)
}

func (m *Model) estimate() error {
	if m.diffData.B.Size() < m.minObservations {
		return ErrInsufficientData
	}

	m.ptCounter = 0

	// Check stationarity
	if !m.checkStationarity() {
		return ErrNonStationarity
	}

	// Estimate parameters based on selected method
	var err error
	switch m.method {
	case MaximumLikelihood:
		err = m.estimateByMaximumLikelihood()
	case ConditionalLeastSquares:
		err = m.estimateByConditionalLS()
	case UnconditionalLeastSquares:
		err = m.estimateByUnconditionalLS()
	case YuleWalker:
		err = m.estimateByYuleWalker()
	default:
		err = m.estimateByMaximumLikelihood()
	}

	if err != nil {
		return err
	}

	// Calculate residuals and diagnostics
	m.calculateResiduals()
	m.calculateDiagnostics()

	m.estimated = true
	return nil
}

func (m *Model) checkStationarity() bool {
	if m.diffData.B.Size() < 20 {
		return true // Assume stationary for small samples
	}

	// Get series in order
	series := m.getDiffSeriesInOrder()

	// Calculate first-order autocorrelation
	mean := m.diffData.Mean()
	var numerator, denominator fixed.Point

	for i := 1; i < len(series); i++ {
		lag0 := series[i-1].Sub(mean)
		lag1 := series[i].Sub(mean)
		numerator = numerator.Add(lag0.Mul(lag1))
		denominator = denominator.Add(lag0.Mul(lag0))
	}

	if denominator.Gt(fixed.Zero) {
		rho := numerator.Div(denominator)
		// Simple stationarity check: |rho| < threshold
		return rho.Abs().Lt(fixed.FromFloat(StationarityThreshold))
	}

	return true
}

func (m *Model) estimateByMaximumLikelihood() error {
	// Initialize parameters
	m.initializeParameters()

	// Newton-Raphson optimization
	for iter := 0; iter < MaxIterations; iter++ {
		oldLL := m.logLikelihood()

		// Calculate gradient and Hessian
		gradient := m.calculateGradient()
		hessian := m.calculateHessian()

		// Update parameters
		delta := solveLinearSystem(hessian, gradient)
		if delta == nil {
			return ErrNumericalInstability
		}

		m.updateParameters(delta)

		// Check convergence
		newLL := m.logLikelihood()
		if m.hasConverged(oldLL, newLL) {
			m.diagnostics.ConvergenceCode = 0
			m.diagnostics.Iterations = iter + 1
			return nil
		}
	}

	m.diagnostics.ConvergenceCode = 1
	return ErrConvergenceFailed
}

func (m *Model) estimateByConditionalLS() error {
	n := m.diffData.B.Size()
	if n <= m.p+m.q {
		return ErrInsufficientData
	}

	// Build design matrix
	startIdx := max(m.p, m.q)
	numObs := n - startIdx

	// Number of parameters
	numParams := m.p + m.q
	if m.includeConstant {
		numParams++
	}

	X := make([][]fixed.Point, numObs)
	y := make([]fixed.Point, numObs)

	// Get data in proper order
	diffSeries := m.getDiffSeriesInOrder()

	for i := uint(0); i < numObs; i++ {
		X[i] = make([]fixed.Point, numParams)
		obsIdx := startIdx + i

		// Dependent variable
		y[i] = diffSeries[obsIdx]

		paramIdx := uint(0)

		// Constant term
		if m.includeConstant {
			X[i][paramIdx] = fixed.One
			paramIdx++
		}

		// AR terms
		for j := uint(1); j <= m.p; j++ {
			if obsIdx >= j {
				X[i][paramIdx] = diffSeries[obsIdx-j]
				paramIdx++
			}
		}

		// MA terms (initially zero for conditional LS)
		for j := uint(0); j < m.q; j++ {
			X[i][paramIdx] = fixed.Zero
			paramIdx++
		}
	}

	// Solve normal equations
	params := solveNormalEquations(X, y)
	if params == nil {
		return ErrNumericalInstability
	}

	m.setParametersFromVector(params)

	// Estimate variance from residuals
	residuals := m.calculateCurrentResiduals()
	if len(residuals) > 0 {
		var sumSquares fixed.Point
		for _, r := range residuals {
			sumSquares = sumSquares.Add(r.Mul(r))
		}
		m.variance = sumSquares.DivInt(len(residuals))
	}

	return nil
}

func (m *Model) estimateByUnconditionalLS() error {
	// For now, fall back to conditional LS
	return m.estimateByConditionalLS()
}

func (m *Model) estimateByYuleWalker() error {
	if m.q > 0 {
		return errors.New("Yule-Walker method only applicable to pure AR models")
	}

	return m.estimateARByYuleWalker()
}

func (m *Model) estimateARByYuleWalker() error {
	n := m.diffData.B.Size()
	if n <= m.p {
		return ErrInsufficientData
	}

	// Calculate autocovariances
	gamma := make([]fixed.Point, m.p+1)
	mean := m.diffData.Mean()

	// Get series in order
	series := m.getDiffSeriesInOrder()

	for lag := uint(0); lag <= m.p; lag++ {
		var covariance fixed.Point
		count := n - lag

		for i := uint(0); i < count; i++ {
			val1 := series[i].Sub(mean)
			val2 := series[i+lag].Sub(mean)
			covariance = covariance.Add(val1.Mul(val2))
		}

		if count > 0 {
			gamma[lag] = covariance.DivInt(int(count))
		}
	}

	// Solve Yule-Walker equations
	return m.solveYuleWalkerEquations(gamma)
}

func (m *Model) solveYuleWalkerEquations(gamma []fixed.Point) error {
	if len(gamma) < 2 || gamma[0].Lte(fixed.Zero) {
		return ErrNumericalInstability
	}

	// Levinson-Durbin algorithm
	phi := make([]fixed.Point, m.p)

	// Initialize
	if m.p >= 1 {
		phi[0] = gamma[1].Div(gamma[0])
		var sigma2 = gamma[0].Mul(fixed.One.Sub(phi[0].Mul(phi[0])))

		// Recursive steps
		for k := uint(2); k <= m.p; k++ {
			// Calculate reflection coefficient
			var sum fixed.Point
			for j := uint(1); j < k; j++ {
				if int(k-j) < len(gamma) && j-1 < uint(len(phi)) {
					sum = sum.Add(phi[j-1].Mul(gamma[k-j]))
				}
			}

			if sigma2.Lte(fixed.Zero) {
				return ErrNumericalInstability
			}

			phiKK := gamma[k].Sub(sum).Div(sigma2)

			// Update coefficients
			phiNew := make([]fixed.Point, k)
			for j := uint(1); j < k; j++ {
				if j-1 < uint(len(phi)) && k-j-1 < uint(len(phi)) {
					phiNew[j-1] = phi[j-1].Sub(phiKK.Mul(phi[k-j-1]))
				}
			}
			phiNew[k-1] = phiKK

			// Copy back
			for j := uint(0); j < k && j < m.p; j++ {
				if j < uint(len(phiNew)) {
					phi[j] = phiNew[j]
				}
			}

			// Update variance
			sigma2 = sigma2.Mul(fixed.One.Sub(phiKK.Mul(phiKK)))
		}

		m.variance = sigma2
	}

	// Set AR parameters
	copy(m.arParams, phi)

	return nil
}

func (m *Model) initializeParameters() {
	// Initialize AR parameters using Yule-Walker if possible
	if m.p > 0 {
		m.estimateARByYuleWalker()
	}

	// Initialize MA parameters to small values
	for i := uint(0); i < m.q; i++ {
		// Small initial values to ensure invertibility
		initVal := fixed.FromFloat(0.1 * (0.5 - float64(i%2)))
		if i < uint(len(m.maParams)) {
			m.maParams[i] = initVal
		}
	}

	// Initialize constant
	if m.includeConstant {
		m.constant = m.diffData.Mean()
	} else {
		m.constant = fixed.Zero
	}

	// Initialize variance if not already set
	if m.variance.Eq(fixed.Zero) {
		m.variance = m.diffData.Variance()
		if m.variance.Lte(fixed.Zero) {
			m.variance = fixed.One
		}
	}
}

func (m *Model) logLikelihood() fixed.Point {
	n := m.diffData.B.Size()
	if n == 0 || m.variance.Lte(fixed.Zero) {
		return fixed.FromFloat(-math.Inf(1))
	}

	// Calculate residuals
	residuals := m.calculateCurrentResiduals()
	if len(residuals) == 0 {
		return fixed.FromFloat(-math.Inf(1))
	}

	// Log-likelihood for Gaussian errors
	var sumSquares fixed.Point
	for _, r := range residuals {
		sumSquares = sumSquares.Add(r.Mul(r))
	}

	nf := fixed.FromUint(uint64(len(residuals)), 0)
	ll := nf.Mul(fixed.FromFloat(-0.5 * math.Log(2*math.Pi)))
	ll = ll.Sub(nf.Mul(m.variance.Log()).DivInt(2))
	ll = ll.Sub(sumSquares.Div(m.variance.MulInt(2)))

	return ll
}

func (m *Model) calculateCurrentResiduals() []fixed.Point {
	n := m.diffData.B.Size()
	if n <= max(m.p, m.q) {
		return []fixed.Point{}
	}

	startIdx := max(m.p, m.q)
	residuals := make([]fixed.Point, n-startIdx)

	mean := m.diffData.Mean()
	series := m.getDiffSeriesInOrder()

	for i := startIdx; i < n; i++ {
		fitted := m.constant
		if !m.includeConstant {
			fitted = fixed.Zero
		}

		// AR component
		for j := uint(1); j <= m.p && j <= uint(i); j++ {
			if j-1 < uint(len(m.arParams)) {
				arCoeff := m.arParams[j-1]
				laggedValue := series[i-j]
				if m.includeConstant {
					laggedValue = laggedValue.Sub(mean)
				}
				fitted = fitted.Add(arCoeff.Mul(laggedValue))
			}
		}

		// MA component (using previously calculated residuals)
		for j := uint(1); j <= m.q && j <= uint(i-startIdx); j++ {
			if j-1 < uint(len(m.maParams)) {
				maCoeff := m.maParams[j-1]
				laggedResidual := residuals[i-startIdx-j]
				fitted = fitted.Add(maCoeff.Mul(laggedResidual))
			}
		}

		if m.includeConstant {
			fitted = fitted.Add(mean)
		}

		actual := series[i]
		residuals[i-startIdx] = actual.Sub(fitted)
	}

	return residuals
}

func (m *Model) calculateGradient() []fixed.Point {
	numParams := m.getParameterCount()
	gradient := make([]fixed.Point, numParams)

	epsilon := fixed.FromFloat(1e-6)
	params := m.getParameterVector()

	for i := 0; i < numParams; i++ {
		original := params[i]

		// Forward difference
		params[i] = original.Add(epsilon)
		m.setParametersFromVector(params)
		forwardLL := m.logLikelihood()

		// Backward difference
		params[i] = original.Sub(epsilon)
		m.setParametersFromVector(params)
		backwardLL := m.logLikelihood()

		// Central difference
		gradient[i] = forwardLL.Sub(backwardLL).Div(epsilon.MulInt(2))

		// Restore original
		params[i] = original
	}

	m.setParametersFromVector(params)
	return gradient
}

func (m *Model) calculateHessian() [][]fixed.Point {
	numParams := m.getParameterCount()
	hessian := make([][]fixed.Point, numParams)
	for i := 0; i < numParams; i++ {
		hessian[i] = make([]fixed.Point, numParams)
	}

	epsilon := fixed.FromFloat(1e-4)
	params := m.getParameterVector()

	// Calculate diagonal elements only (simplified)
	for i := 0; i < numParams; i++ {
		original := params[i]

		// f(x+h)
		params[i] = original.Add(epsilon)
		m.setParametersFromVector(params)
		forward := m.logLikelihood()

		// f(x-h)
		params[i] = original.Sub(epsilon)
		m.setParametersFromVector(params)
		backward := m.logLikelihood()

		// f(x)
		params[i] = original
		m.setParametersFromVector(params)
		center := m.logLikelihood()

		// Second derivative
		hessian[i][i] = forward.Sub(center.MulInt(2)).Add(backward).Div(epsilon.Mul(epsilon))
	}

	return hessian
}

func (m *Model) getParameterCount() int {
	count := int(m.p + m.q)
	if m.includeConstant {
		count++
	}
	return count
}

func (m *Model) getParameterVector() []fixed.Point {
	numParams := m.getParameterCount()
	params := make([]fixed.Point, numParams)

	idx := 0

	if m.includeConstant {
		params[idx] = m.constant
		idx++
	}

	// AR parameters
	for i := 0; i < len(m.arParams) && idx < numParams; i++ {
		params[idx] = m.arParams[i]
		idx++
	}

	// MA parameters
	for i := 0; i < len(m.maParams) && idx < numParams; i++ {
		params[idx] = m.maParams[i]
		idx++
	}

	return params
}

func (m *Model) setParametersFromVector(params []fixed.Point) {
	idx := 0

	if m.includeConstant && idx < len(params) {
		m.constant = params[idx]
		idx++
	}

	// AR parameters
	for i := 0; i < len(m.arParams) && idx < len(params); i++ {
		m.arParams[i] = params[idx]
		idx++
	}

	// MA parameters
	for i := 0; i < len(m.maParams) && idx < len(params); i++ {
		m.maParams[i] = params[idx]
		idx++
	}
}

func (m *Model) updateParameters(delta []fixed.Point) {
	params := m.getParameterVector()

	// Apply update with step size control
	stepSize := fixed.FromFloat(0.1)

	for i := 0; i < len(params) && i < len(delta); i++ {
		params[i] = params[i].Add(delta[i].Mul(stepSize))
	}

	// Ensure parameter constraints
	m.enforceParameterConstraints(params)
	m.setParametersFromVector(params)
}

func (m *Model) enforceParameterConstraints(params []fixed.Point) {
	idx := 0

	// Skip constant (no constraints)
	if m.includeConstant {
		idx++
	}

	// AR parameters: ensure stationarity
	arSum := fixed.Zero
	for i := uint(0); i < m.p && idx < len(params); i++ {
		// Bound individual AR coefficients
		if params[idx].Gt(fixed.FromFloat(0.99)) {
			params[idx] = fixed.FromFloat(0.99)
		} else if params[idx].Lt(fixed.FromFloat(-0.99)) {
			params[idx] = fixed.FromFloat(-0.99)
		}
		arSum = arSum.Add(params[idx].Abs())
		idx++
	}

	// If sum of absolute AR coefficients > 1, scale them down
	if arSum.Gt(fixed.FromFloat(0.99)) {
		scale := fixed.FromFloat(0.99).Div(arSum)
		arIdx := 0
		if m.includeConstant {
			arIdx = 1
		}
		for i := uint(0); i < m.p && arIdx < len(params); i++ {
			params[arIdx] = params[arIdx].Mul(scale)
			arIdx++
		}
	}

	// MA parameters: ensure invertibility
	maSum := fixed.Zero
	for i := uint(0); i < m.q && idx < len(params); i++ {
		// Bound individual MA coefficients
		if params[idx].Gt(fixed.FromFloat(0.99)) {
			params[idx] = fixed.FromFloat(0.99)
		} else if params[idx].Lt(fixed.FromFloat(-0.99)) {
			params[idx] = fixed.FromFloat(-0.99)
		}
		maSum = maSum.Add(params[idx].Abs())
		idx++
	}

	// If sum of absolute MA coefficients > 1, scale them down
	if maSum.Gt(fixed.FromFloat(0.99)) {
		scale := fixed.FromFloat(0.99).Div(maSum)
		maIdx := int(m.p)
		if m.includeConstant {
			maIdx++
		}
		for i := uint(0); i < m.q && maIdx < len(params); i++ {
			params[maIdx] = params[maIdx].Mul(scale)
			maIdx++
		}
	}
}

func (m *Model) hasConverged(oldLL, newLL fixed.Point) bool {
	if newLL.Lt(oldLL) {
		return false
	}

	diff := newLL.Sub(oldLL).Abs()
	tolerance := fixed.FromFloat(ConvergenceTolerance)

	return diff.Lt(tolerance)
}

// Private Methods - Diagnostics

func (m *Model) calculateResiduals() {
	residuals := m.calculateCurrentResiduals()

	// Clear and populate residuals buffer
	m.residuals.Clear()
	for _, r := range residuals {
		m.residuals.PushUpdate(r)
	}

	// Calculate standardized residuals
	if len(residuals) > 0 {
		var sumSquares fixed.Point
		for _, r := range residuals {
			sumSquares = sumSquares.Add(r.Mul(r))
		}

		residualStdDev := fixed.FromFloat(math.Sqrt(sumSquares.DivInt(len(residuals)).Float64()))

		m.standardizedResiduals.Clear()
		for _, r := range residuals {
			if residualStdDev.Gt(fixed.Zero) {
				standardized := r.Div(residualStdDev)
				m.standardizedResiduals.PushUpdate(standardized)
			}
		}
	}
}

func (m *Model) calculateDiagnostics() {
	residuals := m.calculateCurrentResiduals()
	numResiduals := len(residuals)

	if numResiduals == 0 {
		return
	}

	// Log-likelihood
	m.diagnostics.LogLikelihood = m.logLikelihood()

	// Information criteria
	numParams := fixed.FromUint(uint64(m.getParameterCount()), 0)
	nf := fixed.FromUint(uint64(numResiduals), 0)

	// AIC = -2*log(L) + 2*k
	m.diagnostics.AIC = m.diagnostics.LogLikelihood.MulInt(-2).Add(numParams.MulInt(2))

	// BIC = -2*log(L) + k*log(n)
	m.diagnostics.BIC = m.diagnostics.LogLikelihood.MulInt(-2).Add(numParams.Mul(nf.Log()))

	// AICc = AIC + 2*k*(k+1)/(n-k-1)
	if numResiduals > m.getParameterCount()+1 {
		correction := numParams.Mul(numParams.Add(fixed.One)).MulInt(2)
		correction = correction.Div(nf.Sub(numParams).Sub(fixed.One))
		m.diagnostics.AICC = m.diagnostics.AIC.Add(correction)
	} else {
		m.diagnostics.AICC = m.diagnostics.AIC
	}

	// Calculate error metrics
	var sumSquares, sumAbs, sumPctError fixed.Point
	var validPctCount int

	// Get actual values for MAPE calculation
	series := m.getDiffSeriesInOrder()
	startIdx := max(m.p, m.q)

	for i, r := range residuals {
		sumSquares = sumSquares.Add(r.Mul(r))
		sumAbs = sumAbs.Add(r.Abs())

		// For MAPE
		if int(startIdx)+i < len(series) {
			actual := series[int(startIdx)+i]
			if actual.Abs().Gt(fixed.FromFloat(1e-10)) {
				pctError := r.Abs().Div(actual.Abs()).MulInt(100)
				sumPctError = sumPctError.Add(pctError)
				validPctCount++
			}
		}
	}

	// RMSE
	m.diagnostics.RMSE = fixed.FromFloat(math.Sqrt(sumSquares.DivInt(numResiduals).Float64()))

	// MAE
	m.diagnostics.MAE = sumAbs.DivInt(numResiduals)

	// MAPE
	if validPctCount > 0 {
		m.diagnostics.MAPE = sumPctError.DivInt(validPctCount)
	}

	// Ljung-Box test
	m.diagnostics.LjungBoxPValue = m.ljungBoxTest(residuals)

	// Jarque-Bera test for normality
	m.diagnostics.JarqueBeraTest = m.jarqueBeraTest(residuals)

	// Stationarity
	m.diagnostics.IsStationary = m.checkStationarity()
}

func (m *Model) ljungBoxTest(residuals []fixed.Point) fixed.Point {
	n := len(residuals)
	if n < 10 {
		return fixed.One
	}

	maxLag := min(10, n/4)
	var testStat fixed.Point

	// Calculate mean
	mean := fixed.Zero
	for _, r := range residuals {
		mean = mean.Add(r)
	}
	mean = mean.DivInt(n)

	// Calculate variance
	var variance fixed.Point
	for _, r := range residuals {
		diff := r.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.DivInt(n)

	if variance.Lte(fixed.Zero) {
		return fixed.One
	}

	// Calculate autocorrelations
	for lag := 1; lag <= maxLag; lag++ {
		var autocovariance fixed.Point
		count := n - lag

		for i := 0; i < count; i++ {
			val1 := residuals[i].Sub(mean)
			val2 := residuals[i+lag].Sub(mean)
			autocovariance = autocovariance.Add(val1.Mul(val2))
		}

		autocorr := autocovariance.DivInt(count).Div(variance)
		contribution := autocorr.Mul(autocorr).Div(fixed.FromUint(uint64(n-lag), 0))
		testStat = testStat.Add(contribution)
	}

	testStat = testStat.Mul(fixed.FromUint(uint64(n), 0).Mul(fixed.FromUint(uint64(n+2), 0)))

	// Approximate p-value
	if testStat.Lt(fixed.FromFloat(15.51)) {
		return fixed.FromFloat(0.1)
	} else {
		return fixed.FromFloat(0.01)
	}
}

func (m *Model) jarqueBeraTest(residuals []fixed.Point) fixed.Point {
	n := len(residuals)
	if n < 7 {
		return fixed.One
	}

	// Calculate mean
	var mean fixed.Point
	for _, r := range residuals {
		mean = mean.Add(r)
	}
	mean = mean.DivInt(n)

	// Calculate moments
	var m2, m3, m4 fixed.Point
	for _, r := range residuals {
		diff := r.Sub(mean)
		diff2 := diff.Mul(diff)
		diff3 := diff2.Mul(diff)
		diff4 := diff3.Mul(diff)

		m2 = m2.Add(diff2)
		m3 = m3.Add(diff3)
		m4 = m4.Add(diff4)
	}

	m2 = m2.DivInt(n)
	m3 = m3.DivInt(n)
	m4 = m4.DivInt(n)

	if m2.Lte(fixed.Zero) {
		return fixed.One
	}

	// Skewness and kurtosis
	skewness := m3.Div(fixed.FromFloat(math.Pow(m2.Float64(), 1.5)))
	kurtosis := m4.Div(m2.Mul(m2)).Sub(fixed.FromFloat(3.0))

	// JB statistic
	nf := fixed.FromUint(uint64(n), 0)
	jb := nf.Div(fixed.FromFloat(6.0))
	jb = jb.Mul(skewness.Mul(skewness).Add(kurtosis.Mul(kurtosis).Div(fixed.FromFloat(4.0))))

	// Approximate p-value
	if jb.Lt(fixed.FromFloat(5.99)) {
		return fixed.FromFloat(0.1)
	} else {
		return fixed.FromFloat(0.01)
	}
}

func (m *Model) checkParameterValidity() error {
	// Check AR parameter stationarity
	var arSum fixed.Point
	for i := 0; i < len(m.arParams); i++ {
		arSum = arSum.Add(m.arParams[i].Abs())
	}

	if arSum.Gte(fixed.One) {
		return errors.New("AR parameters suggest non-stationarity")
	}

	// Check MA parameter invertibility
	var maSum fixed.Point
	for i := 0; i < len(m.maParams); i++ {
		maSum = maSum.Add(m.maParams[i].Abs())
	}

	if maSum.Gte(fixed.One) {
		return errors.New("MA parameters suggest non-invertibility")
	}

	// Check variance
	if m.variance.Lte(fixed.Zero) {
		return errors.New("invalid variance estimate")
	}

	return nil
}

func (m *Model) checkResidualProperties() error {
	if m.residuals.B.Size() < 10 {
		return nil
	}

	// Check for autocorrelation
	if m.diagnostics.LjungBoxPValue.Lt(fixed.FromFloat(0.05)) {
		return errors.New("residuals show significant autocorrelation")
	}

	return nil
}

// Private Methods - Forecasting

func (m *Model) initializeForecastState() *forecastState {
	state := &forecastState{
		diffSeries:      make([]fixed.Point, 0),
		residuals:       make([]fixed.Point, 0),
		mean:            m.diffData.Mean(),
		variance:        m.variance,
		rawValues:       make([]fixed.Point, 0),
		forecastedDiffs: make([]fixed.Point, 0),
	}

	// Get data in oldest-to-newest order
	if m.diffData.B.Size() > 0 {
		state.diffSeries = m.getDiffSeriesInOrder()
	}

	if m.residuals.B.Size() > 0 {
		residualData := m.residuals.B.Data()
		// Reverse to get oldest to newest
		for i := len(residualData) - 1; i >= 0; i-- {
			state.residuals = append(state.residuals, residualData[i])
		}
	}

	// Store raw values for undifferencing
	if m.rawData.B.Size() > 0 {
		state.rawValues = m.getRawSeriesInOrder()
	}

	return state
}

func (m *Model) forecastOneStep(state *forecastState, step uint) (ForecastResult, error) {
	var result ForecastResult

	// Point forecast in differenced scale
	forecast := fixed.Zero

	// Add constant term if included
	if m.includeConstant {
		forecast = m.constant
	}

	// AR component
	for i := uint(0); i < m.p && i < uint(len(m.arParams)); i++ {
		arCoeff := m.arParams[i]

		// Get the appropriate lagged value
		var laggedValue fixed.Point
		totalDiffSeries := len(state.diffSeries) + len(state.forecastedDiffs)
		lagIdx := totalDiffSeries - 1 - int(i)

		if lagIdx >= len(state.diffSeries) {
			// Use forecasted differences
			forecastIdx := lagIdx - len(state.diffSeries)
			if forecastIdx < len(state.forecastedDiffs) {
				laggedValue = state.forecastedDiffs[forecastIdx]
			}
		} else if lagIdx >= 0 {
			// Use historical differences
			laggedValue = state.diffSeries[lagIdx]
		}

		if m.includeConstant {
			laggedValue = laggedValue.Sub(state.mean)
		}
		forecast = forecast.Add(arCoeff.Mul(laggedValue))
	}

	// MA component
	for i := uint(0); i < m.q && i < uint(len(m.maParams)); i++ {
		maCoeff := m.maParams[i]

		// For forecasting, we use actual residuals for past values
		// and assume zero residuals for future values
		if int(i) < len(state.residuals) {
			lagIdx := len(state.residuals) - 1 - int(i)
			if lagIdx >= 0 {
				laggedResidual := state.residuals[lagIdx]
				forecast = forecast.Add(maCoeff.Mul(laggedResidual))
			}
		}
	}

	// Add mean if constant is included
	if m.includeConstant {
		forecast = forecast.Add(state.mean)
	}

	// Store the forecasted difference
	state.forecastedDiffs = append(state.forecastedDiffs, forecast)

	// Convert back to original scale
	originalForecast := m.undifferenceWithState(forecast, state, step)

	// Calculate forecast variance and confidence intervals
	forecastVar := m.calculateForecastVariance(step + 1)
	standardError := fixed.FromFloat(math.Sqrt(forecastVar.Float64()))

	result.PointForecast = originalForecast
	result.StandardError = standardError

	// Confidence intervals (normal approximation)
	z95 := fixed.FromFloat(1.96)
	z80 := fixed.FromFloat(1.282)

	margin95 := z95.Mul(standardError)
	margin80 := z80.Mul(standardError)

	result.ConfidenceInterval.Lower95 = originalForecast.Sub(margin95)
	result.ConfidenceInterval.Upper95 = originalForecast.Add(margin95)
	result.ConfidenceInterval.Lower80 = originalForecast.Sub(margin80)
	result.ConfidenceInterval.Upper80 = originalForecast.Add(margin80)

	// Prediction intervals (include model uncertainty)
	predVar := forecastVar.Add(m.variance)
	predStdErr := fixed.FromFloat(math.Sqrt(predVar.Float64()))

	predMargin95 := z95.Mul(predStdErr)
	predMargin80 := z80.Mul(predStdErr)

	result.PredictionInterval.Lower95 = originalForecast.Sub(predMargin95)
	result.PredictionInterval.Upper95 = originalForecast.Add(predMargin95)
	result.PredictionInterval.Lower80 = originalForecast.Sub(predMargin80)
	result.PredictionInterval.Upper80 = originalForecast.Add(predMargin80)

	return result, nil
}

func (m *Model) calculateForecastVariance(step uint) fixed.Point {
	if step == 0 {
		return m.variance
	}

	// Calculate psi weights up to step h-1
	psiWeights := m.calculatePsiWeights(step)

	// Sum of squared psi weights
	sumSquaredPsi := fixed.One // psi_0 = 1
	for i := uint(1); i < step && i-1 < uint(len(psiWeights)); i++ {
		psi := psiWeights[i-1]
		sumSquaredPsi = sumSquaredPsi.Add(psi.Mul(psi))
	}

	return m.variance.Mul(sumSquaredPsi)
}

func (m *Model) calculatePsiWeights(maxLag uint) []fixed.Point {
	if maxLag == 0 {
		return []fixed.Point{}
	}

	psi := make([]fixed.Point, maxLag)

	for j := uint(0); j < maxLag; j++ {
		psiJ := fixed.Zero

		// AR contribution
		for i := uint(0); i < m.p && i <= j && i < uint(len(m.arParams)); i++ {
			if i == 0 && j > 0 {
				psiJ = psiJ.Add(m.arParams[i].Mul(psi[j-1]))
			} else if j >= i && i > 0 {
				psiJ = psiJ.Add(m.arParams[i].Mul(psi[j-i]))
			}
		}

		// MA contribution (only for j < q)
		if j < m.q && j < uint(len(m.maParams)) {
			psiJ = psiJ.Add(m.maParams[j])
		}

		psi[j] = psiJ
	}

	return psi
}

func (m *Model) updateForecastState(state *forecastState, forecast fixed.Point, step uint) {
	// For residuals, we assume zero for forecasted periods
	state.residuals = append(state.residuals, fixed.Zero)
}

// Private Helper Methods

func (m *Model) getDiffSeriesInOrder() []fixed.Point {
	if m.diffData.B.Size() == 0 {
		return []fixed.Point{}
	}

	data := m.diffData.B.Data()
	result := make([]fixed.Point, len(data))

	// Reverse the data since Get(0) returns most recent
	for i := 0; i < len(data); i++ {
		result[i] = data[len(data)-1-i]
	}

	return result
}

func (m *Model) getRawSeriesInOrder() []fixed.Point {
	if m.rawData.B.Size() == 0 {
		return []fixed.Point{}
	}

	data := m.rawData.B.Data()
	result := make([]fixed.Point, len(data))

	// Reverse the data
	for i := 0; i < len(data); i++ {
		result[i] = data[len(data)-1-i]
	}

	return result
}

// Package-level Helper Functions

func solveLinearSystem(A [][]fixed.Point, b []fixed.Point) []fixed.Point {
	n := len(b)
	if n == 0 || len(A) != n {
		return nil
	}

	// Create augmented matrix
	augmented := make([][]fixed.Point, n)
	for i := 0; i < n; i++ {
		augmented[i] = make([]fixed.Point, n+1)
		for j := 0; j < n; j++ {
			augmented[i][j] = A[i][j]
		}
		augmented[i][n] = b[i]
	}

	// Gaussian elimination with partial pivoting
	for k := 0; k < n-1; k++ {
		// Find pivot
		maxRow := k
		for i := k + 1; i < n; i++ {
			if augmented[i][k].Abs().Gt(augmented[maxRow][k].Abs()) {
				maxRow = i
			}
		}

		// Swap rows
		if maxRow != k {
			augmented[k], augmented[maxRow] = augmented[maxRow], augmented[k]
		}

		// Check for singular matrix
		if augmented[k][k].Abs().Lt(fixed.FromFloat(1e-12)) {
			return nil
		}

		// Eliminate column
		for i := k + 1; i < n; i++ {
			factor := augmented[i][k].Div(augmented[k][k])
			for j := k; j <= n; j++ {
				augmented[i][j] = augmented[i][j].Sub(factor.Mul(augmented[k][j]))
			}
		}
	}

	// Back substitution
	x := make([]fixed.Point, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = augmented[i][n]
		for j := i + 1; j < n; j++ {
			x[i] = x[i].Sub(augmented[i][j].Mul(x[j]))
		}
		if augmented[i][i].Abs().Lt(fixed.FromFloat(1e-12)) {
			return nil
		}
		x[i] = x[i].Div(augmented[i][i])
	}

	return x
}

func solveNormalEquations(X [][]fixed.Point, y []fixed.Point) []fixed.Point {
	if len(X) == 0 || len(X[0]) == 0 {
		return nil
	}

	rows := len(X)
	cols := len(X[0])

	// Compute X'X
	XtX := make([][]fixed.Point, cols)
	for i := 0; i < cols; i++ {
		XtX[i] = make([]fixed.Point, cols)
		for j := 0; j < cols; j++ {
			var sum fixed.Point
			for k := 0; k < rows; k++ {
				sum = sum.Add(X[k][i].Mul(X[k][j]))
			}
			XtX[i][j] = sum
		}
	}

	// Compute X'y
	Xty := make([]fixed.Point, cols)
	for i := 0; i < cols; i++ {
		var sum fixed.Point
		for k := 0; k < rows; k++ {
			sum = sum.Add(X[k][i].Mul(y[k]))
		}
		Xty[i] = sum
	}

	// Solve (X'X)β = X'y
	return solveLinearSystem(XtX, Xty)
}

func binomialCoefficient(n, k uint) fixed.Point {
	if k > n {
		return fixed.Zero
	}
	if k == 0 || k == n {
		return fixed.One
	}

	// Use multiplicative formula to avoid overflow
	result := fixed.One
	for i := uint(0); i < k; i++ {
		result = result.MulInt(int(n - i))
		result = result.DivInt(int(i + 1))
	}
	return result
}
