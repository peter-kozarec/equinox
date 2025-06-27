package simulation

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"

	"math/rand"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

var (
	pointFive = fixed.New(5, 1)
)

type MonteCarloExecutor struct {
	logger    *zap.Logger
	simulator *Simulator
	rng       *rand.Rand

	startTime  time.Time
	startPrice fixed.Point
	baseSpread fixed.Point
	mu         fixed.Point
	sigma      fixed.Point
	deltaT     fixed.Point
	steps      int64
	t          int64

	// Tick timing parameters
	avgTickInterval time.Duration
	tickVariability float64

	// Volume parameters
	avgVolume      fixed.Point
	volumeVariance float64

	// Price model parameters
	deltaLogPre1 fixed.Point
	deltaLogPre2 fixed.Point

	// Spread dynamics
	spreadVolatility float64
	minSpread        fixed.Point
	maxSpread        fixed.Point

	lastTime      time.Time
	lastPrice     fixed.Point
	currentSpread fixed.Point

	tick common.Tick

	normPriceDigits  int
	normVolumeDigits int
}

func NewMonteCarloExecutor(
	logger *zap.Logger,
	sim *Simulator,
	rng *rand.Rand,
	startTime time.Time,
	startPrice, fullSpread, mu, sigma, deltaT fixed.Point,
	steps int64) *MonteCarloExecutor {

	avgTickInterval := time.Duration(333_000_000) // ~333ms default

	return &MonteCarloExecutor{
		logger:    logger,
		simulator: sim,
		rng:       rng,

		startTime:  startTime,
		startPrice: startPrice,
		baseSpread: fullSpread.DivInt(2), // Half spread for bid/ask calculation
		mu:         mu,
		sigma:      sigma,
		deltaT:     deltaT,
		steps:      steps,

		// Tick timing - realistic microsecond-level variations
		avgTickInterval: avgTickInterval,
		tickVariability: 0.3, // 30% variation in tick timing

		// Volume parameters - more realistic volume simulation
		avgVolume:      fixed.New(100, 0), // Average 100 units
		volumeVariance: 0.5,               // 50% variance in volume

		// Spread dynamics
		spreadVolatility: 0.1,
		minSpread:        fullSpread.Mul(fixed.New(5, 1)),  // 50% of base spread
		maxSpread:        fullSpread.Mul(fixed.New(15, 1)), // 150% of base spread

		// Pre-calculated values for GBM
		deltaLogPre1: mu.Sub(sigma.Mul(sigma).Mul(pointFive)).Mul(deltaT),
		deltaLogPre2: sigma.Mul(deltaT.Sqrt()),

		lastTime:      startTime,
		lastPrice:     startPrice,
		currentSpread: fullSpread.DivInt(2),
	}
}
func NewEurUsdMonteCarloTickSimulator(
	logger *zap.Logger,
	simulator *Simulator,
	rng *rand.Rand,
	duration time.Duration,
	mu, sigma float64) *MonteCarloExecutor {

	// EURUSD-specific configuration
	const (
		// Market characteristics
		eurUsdStartPrice    = 1.0550  // Typical EURUSD starting price
		eurUsdTypicalSpread = 0.00003 // 0.3 pips spread
		eurUsdMinSpread     = 0.00001 // 0.1 pips minimum
		eurUsdMaxSpread     = 0.00006 // 0.6 pips maximum

		// Tick timing (realistic for EURUSD)
		avgTickIntervalSeconds = 1    // second average between ticks
		tickTimingVariability  = 0.45 // 45% timing variation

		// Volume characteristics
		avgVolumeUnits    = 1    // 1 units average volume
		volumeVariability = 0.65 // 65% volume variance

		// Spread dynamics
		spreadVolatility = 0.12 // 12% spread volatility

		// Normalization digits
		normPriceDigits  = 5
		normVolumeDigits = 2
	)

	// Setup timing
	startTime := time.Now()

	// Convert duration to number of ticks
	totalSeconds := int64(duration.Seconds())
	avgTickInterval := time.Duration(avgTickIntervalSeconds * float64(time.Second))
	estimatedTicks := totalSeconds / int64(avgTickIntervalSeconds)

	// Time delta for price model (convert to fraction of year)
	secondsPerYear := 365.25 * 24 * 3600
	deltaT := fixed.FromFloat(avgTickIntervalSeconds / secondsPerYear)

	// Convert price and spread to fixed point
	startPrice := fixed.FromFloat(eurUsdStartPrice)
	fullSpread := fixed.FromFloat(eurUsdTypicalSpread)
	minSpread := fixed.FromFloat(eurUsdMinSpread)
	maxSpread := fixed.FromFloat(eurUsdMaxSpread)

	// Convert mu and sigma to fixed point
	muFixed := fixed.FromFloat(mu)
	sigmaFixed := fixed.FromFloat(sigma)

	// Create the base Monte Carlo executor
	executor := NewMonteCarloExecutor(
		logger,
		simulator,
		rng,
		startTime,
		startPrice,
		fullSpread,
		muFixed,
		sigmaFixed,
		deltaT,
		estimatedTicks,
	)

	// Configure EURUSD-specific tick parameters
	executor.SetTickParameters(
		avgTickInterval,
		tickTimingVariability,
		fixed.New(int64(avgVolumeUnits), 0),
		volumeVariability,
	)

	// Configure EURUSD-specific spread dynamics
	executor.SetSpreadDynamics(
		spreadVolatility,
		minSpread,
		maxSpread,
	)

	executor.normPriceDigits = normPriceDigits
	executor.normVolumeDigits = normVolumeDigits

	// Log configuration
	logger.Debug("EURUSD Monte Carlo Tick Simulator configured",
		zap.Duration("duration", duration),
		zap.Float64("mu_annual", mu),
		zap.Float64("sigma_annual", sigma),
		zap.Float64("start_price", eurUsdStartPrice),
		zap.Float64("avg_spread_pips", eurUsdTypicalSpread*100000),
		zap.Float64("avg_tick_interval_sec", avgTickIntervalSeconds),
		zap.Int64("estimated_ticks", estimatedTicks),
		zap.Time("start_time", startTime),
	)

	return executor
}

// SetTickParameters allows customization of tick characteristics
func (e *MonteCarloExecutor) SetTickParameters(
	avgInterval time.Duration,
	intervalVariability float64,
	avgVol fixed.Point,
	volVariance float64) {

	e.avgTickInterval = avgInterval
	e.tickVariability = intervalVariability
	e.avgVolume = avgVol
	e.volumeVariance = volVariance
}

// SetSpreadDynamics configures dynamic spread behavior
func (e *MonteCarloExecutor) SetSpreadDynamics(
	volatility float64,
	minSpread, maxSpread fixed.Point) {

	e.spreadVolatility = volatility
	e.minSpread = minSpread
	e.maxSpread = maxSpread
}

func (e *MonteCarloExecutor) DoOnce() error {
	if e.t >= e.steps {
		return mapper.ErrEof
	}

	// Generate next price using Geometric Brownian Motion
	z := e.rng.NormFloat64()
	deltaLog := e.deltaLogPre1.Add(e.deltaLogPre2.Mul(fixed.FromFloat(z)))
	e.lastPrice = e.lastPrice.Mul(deltaLog.Exp())

	// Dynamic spread based on volatility
	e.updateSpread()

	// Variable tick timing - more realistic than fixed intervals
	tickInterval := e.generateTickInterval()
	e.lastTime = e.lastTime.Add(tickInterval)
	e.t++

	// Generate realistic volumes
	askVol, bidVol := e.generateVolumes()

	// Build tick with realistic bid/ask spread
	e.tick.Ask = e.lastPrice.Add(e.currentSpread)
	e.tick.Bid = e.lastPrice.Sub(e.currentSpread)
	e.tick.TimeStamp = e.lastTime.UnixNano()
	e.tick.AskVolume = askVol
	e.tick.BidVolume = bidVol

	// Optional: Add some tick-level noise for more realism
	e.addTickNoise()

	e.tick.Ask = e.tick.Ask.Rescale(e.normPriceDigits)
	e.tick.Bid = e.tick.Bid.Rescale(e.normPriceDigits)

	e.tick.AskVolume = e.tick.AskVolume.Rescale(e.normVolumeDigits)
	e.tick.BidVolume = e.tick.BidVolume.Rescale(e.normVolumeDigits)

	if err := e.simulator.OnTick(e.tick); err != nil {
		return err
	}

	return nil
}

// updateSpread simulates dynamic spread changes based on market conditions
func (e *MonteCarloExecutor) updateSpread() {
	if e.spreadVolatility <= 0 {
		return
	}

	// Spread tends to widen during high volatility
	spreadChange := e.rng.NormFloat64() * e.spreadVolatility
	newSpread := e.currentSpread.Mul(fixed.FromFloat(1.0 + spreadChange))

	// Clamp spread within bounds
	if newSpread.Lt(e.minSpread) {
		e.currentSpread = e.minSpread
	} else if newSpread.Gt(e.maxSpread) {
		e.currentSpread = e.maxSpread
	} else {
		e.currentSpread = newSpread
	}
}

// generateTickInterval creates realistic variable tick timing
func (e *MonteCarloExecutor) generateTickInterval() time.Duration {
	if e.tickVariability <= 0 {
		return e.avgTickInterval
	}

	// Use exponential distribution for more realistic tick intervals
	// This creates clustering and gaps similar to real market data
	lambda := 1.0 / float64(e.avgTickInterval.Nanoseconds())
	interval := e.rng.ExpFloat64() / lambda

	// Add some bounds to prevent extreme values
	minInterval := float64(e.avgTickInterval.Nanoseconds()) * (1.0 - e.tickVariability)
	maxInterval := float64(e.avgTickInterval.Nanoseconds()) * (1.0 + e.tickVariability*3)

	if interval < minInterval {
		interval = minInterval
	} else if interval > maxInterval {
		interval = maxInterval
	}

	return time.Duration(int64(interval))
}

// generateVolumes creates realistic bid/ask volumes
func (e *MonteCarloExecutor) generateVolumes() (askVol, bidVol fixed.Point) {
	// Generate volumes with log-normal distribution for realism
	askVariation := e.rng.NormFloat64() * e.volumeVariance
	bidVariation := e.rng.NormFloat64() * e.volumeVariance

	askMultiplier := fixed.FromFloat(1.0 + askVariation).Exp()
	bidMultiplier := fixed.FromFloat(1.0 + bidVariation).Exp()

	askVol = e.avgVolume.Mul(askMultiplier)
	bidVol = e.avgVolume.Mul(bidMultiplier)

	// Ensure positive volumes
	if askVol.Lte(fixed.Zero) {
		askVol = fixed.One
	}
	if bidVol.Lte(fixed.Zero) {
		bidVol = fixed.One
	}

	return askVol, bidVol
}

// addTickNoise adds small random variations to simulate market microstructure
func (e *MonteCarloExecutor) addTickNoise() {
	// Small random adjustments to bid/ask to simulate order book dynamics
	tickSize := e.currentSpread.DivInt(10) // Minimum tick size

	askNoise := fixed.FromFloat(e.rng.NormFloat64() * 0.1).Mul(tickSize)
	bidNoise := fixed.FromFloat(e.rng.NormFloat64() * 0.1).Mul(tickSize)

	e.tick.Ask = e.tick.Ask.Add(askNoise)
	e.tick.Bid = e.tick.Bid.Add(bidNoise)

	// Ensure bid < ask
	if e.tick.Bid.Gte(e.tick.Ask) {
		mid := e.tick.Bid.Add(e.tick.Ask).DivInt(2)
		e.tick.Bid = mid.Sub(tickSize)
		e.tick.Ask = mid.Add(tickSize)
	}
}
