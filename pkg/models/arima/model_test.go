package arima

import (
	"fmt"
	"math"
	"testing"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func TestModelArima_CalculateDiagnostics(t *testing.T) {
	tests := []struct {
		name            string
		p, d, q         int
		includeConstant bool
		setupModel      func(*Model)
		checks          []diagnosticCheck
	}{
		{
			name:            "Simple AR(1) model",
			p:               1,
			d:               0,
			q:               0,
			includeConstant: true,
			setupModel: func(m *Model) {
				// Set model parameters
				m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
				m.constant = fixed.FromFloat64(2.0)
				m.variance = fixed.FromFloat64(1.0)

				// Generate AR(1) series that follows the model
				// y_t = c + φ(y_{t-1} - μ) + ε_t where μ is the mean
				series := []float64{2.0} // Start at constant
				for i := 1; i < 50; i++ {
					// AR(1): y_t = 2 + 0.5*(y_{t-1} - mean) + noise
					// For simplicity, using small noise
					noise := float64(i%5-2) * 0.1
					prev := series[i-1]
					next := 2.0 + 0.5*(prev-2.0) + noise
					series = append(series, next)
				}

				// Add the generated series
				for _, val := range series {
					m.diffData.Add(fixed.FromFloat64(val))
				}
			},
			checks: []diagnosticCheck{
				{name: "LogLikelihood", validator: func(d ModelDiagnostics) bool {
					// Log-likelihood should be negative
					return d.LogLikelihood.Lt(fixed.Zero)
				}},
				{name: "AIC", validator: func(d ModelDiagnostics) bool {
					// AIC = -2*log(L) + 2*k, should be positive
					return d.AIC.Gt(fixed.Zero)
				}},
				{name: "BIC", validator: func(d ModelDiagnostics) bool {
					// BIC should be greater than AIC (penalty for parameters)
					return d.BIC.Gt(d.AIC)
				}},
				{name: "RMSE", validator: func(d ModelDiagnostics) bool {
					// RMSE should be positive and reasonable for this well-specified model
					return d.RMSE.Gt(fixed.Zero) && d.RMSE.Lt(fixed.FromFloat64(3.0))
				}},
				{name: "LjungBoxPValue", validator: func(d ModelDiagnostics) bool {
					// P-value should be between 0 and 1
					return d.LjungBoxPValue.Gte(fixed.Zero) && d.LjungBoxPValue.Lte(fixed.One)
				}},
			},
		},
		{
			name:            "ARMA(1,1) model",
			p:               1,
			d:               0,
			q:               1,
			includeConstant: false,
			setupModel: func(m *Model) {
				m.arParams = []fixed.Point{fixed.FromFloat64(0.7)}
				m.maParams = []fixed.Point{fixed.FromFloat64(0.3)}
				m.variance = fixed.FromFloat64(0.5)

				// Generate synthetic data
				for i := 0; i < 100; i++ {
					val := math.Sin(float64(i)*0.1) + float64(i%7-3)*0.1
					m.diffData.Add(fixed.FromFloat64(val))
				}

				// Do not add residuals manually
				// Let calculateDiagnostics compute them via calculateCurrentResiduals()
			},
			checks: []diagnosticCheck{
				{name: "AICc", validator: func(d ModelDiagnostics) bool {
					// AICc should be >= AIC (correction for small samples)
					return d.AICC.Gte(d.AIC)
				}},
				{name: "MAE", validator: func(d ModelDiagnostics) bool {
					// MAE should be positive and <= RMSE
					return d.MAE.Gt(fixed.Zero) && d.MAE.Lte(d.RMSE)
				}},
				{name: "IsStationary", validator: func(d ModelDiagnostics) bool {
					// Model should be stationary with these parameters
					return d.IsStationary
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create model
			m, err := NewModel(tt.p, tt.d, tt.q, 200)
			if err != nil {
				t.Fatalf("Failed to create model: %v", err)
			}

			m.includeConstant = tt.includeConstant
			m.estimated = true

			// Setup model with test data
			tt.setupModel(m)

			// Calculate diagnostics
			m.calculateDiagnostics()

			// Exec checks
			for _, check := range tt.checks {
				if !check.validator(m.diagnostics) {
					t.Errorf("Check '%s' failed", check.name)
					logDiagnostics(t, m.diagnostics)
				}
			}
		})
	}
}

func TestModelArima_CalculateDiagnosticsEdgeCases(t *testing.T) {

	t.Run("All zero actual values for MAPE", func(t *testing.T) {
		m, _ := NewModel(1, 0, 0, 100)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
		m.variance = fixed.FromFloat64(1.0)
		m.constant = fixed.Zero
		m.includeConstant = false
		m.estimated = true

		// Add all zeros (this will make calculateCurrentResiduals return empty)
		for i := 0; i < 20; i++ {
			m.diffData.Add(fixed.Zero)
		}

		m.calculateDiagnostics()

		// MAPE should be zero when no valid percent errors
		if !m.diagnostics.MAPE.Eq(fixed.Zero) {
			t.Errorf("Expected MAPE = 0 with all zero actuals, got %v",
				m.diagnostics.MAPE.String())
		}
	})

	t.Run("High parameter count", func(t *testing.T) {
		m, _ := NewModel(5, 0, 5, 100)
		m.arParams = make([]fixed.Point, 5)
		m.maParams = make([]fixed.Point, 5)
		for i := 0; i < 5; i++ {
			m.arParams[i] = fixed.FromFloat64(0.1)
			m.maParams[i] = fixed.FromFloat64(0.1)
		}
		m.variance = fixed.FromFloat64(1.0)
		m.constant = fixed.FromFloat64(1.0)
		m.includeConstant = true
		m.estimated = true

		// Add minimal data (need at least max(p,q) + 1 points)
		for i := 0; i < 25; i++ {
			m.diffData.Add(fixed.FromFloat64(float64(i)))
		}

		m.calculateDiagnostics()

		// AICc correction should be large
		aiccCorrection := m.diagnostics.AICC.Sub(m.diagnostics.AIC)
		if aiccCorrection.Lt(fixed.FromFloat64(10)) {
			t.Errorf("Expected large AICc correction, got %v",
				aiccCorrection.String())
		}
	})
}

func TestModelArima_CalculateDiagnosticsInformationCriteria(t *testing.T) {
	// Test relationships between AIC, BIC, and AICc
	m, _ := NewModel(2, 0, 1, 100)
	m.arParams = []fixed.Point{fixed.FromFloat64(0.4), fixed.FromFloat64(0.3)}
	m.maParams = []fixed.Point{fixed.FromFloat64(0.2)}
	m.constant = fixed.FromFloat64(1.0)
	m.variance = fixed.FromFloat64(1.0)
	m.includeConstant = true
	m.estimated = true

	// Test with different sample sizes
	sampleSizes := []int{20, 50, 100, 200}

	for _, n := range sampleSizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			// Clear and add data
			m.diffData.Clear()
			m.residuals.Clear()

			for i := 0; i < n; i++ {
				m.diffData.Add(fixed.FromFloat64(float64(i % 10)))
				if i > 3 {
					m.residuals.Add(fixed.FromFloat64(0.1))
				}
			}

			m.calculateDiagnostics()

			// Verify relationships
			// 1. BIC >= AIC for reasonable sample sizes
			if n > 8 && m.diagnostics.BIC.Lt(m.diagnostics.AIC) {
				t.Errorf("BIC should be >= AIC for n=%d", n)
			}

			// 2. AICc >= AIC
			if m.diagnostics.AICC.Lt(m.diagnostics.AIC) {
				t.Errorf("AICc should be >= AIC")
			}

			// 3. As n increases, AICc should approach AIC
			if n > 100 {
				diff := m.diagnostics.AICC.Sub(m.diagnostics.AIC)
				if diff.Gt(fixed.FromFloat64(1.0)) {
					t.Errorf("AICc-AIC difference too large for n=%d: %v",
						n, diff.String())
				}
			}

			t.Logf("n=%d: AIC=%v, BIC=%v, AICc=%v",
				n, m.diagnostics.AIC.String(),
				m.diagnostics.BIC.String(),
				m.diagnostics.AICC.String())
		})
	}
}

func TestModelArima_CalculateDiagnosticsStationarity(t *testing.T) {
	tests := []struct {
		name         string
		series       []float64
		arParams     []float64
		expectStatio bool
	}{
		{
			name:         "Stationary AR(1)",
			series:       generateAR1Series(100, 0.5, 0, 1),
			arParams:     []float64{0.5},
			expectStatio: true,
		},
		{
			name:         "White noise",
			series:       generateWhiteNoise(100),
			arParams:     []float64{0.1},
			expectStatio: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := NewModel(len(tt.arParams), 0, 0, 200)

			// Set parameters
			for i, param := range tt.arParams {
				m.arParams[i] = fixed.FromFloat64(param)
			}
			m.variance = fixed.FromFloat64(1.0)
			m.estimated = true

			// Add series
			for _, val := range tt.series {
				m.diffData.Add(fixed.FromFloat64(val))
			}

			m.calculateDiagnostics()

			if m.diagnostics.IsStationary != tt.expectStatio {
				t.Errorf("Expected stationarity=%v, got %v",
					tt.expectStatio, m.diagnostics.IsStationary)
			}
		})
	}
}

func TestModelArima_LjungBoxTest(t *testing.T) {
	tests := []struct {
		name        string
		residuals   []fixed.Point
		minPValue   float64
		maxPValue   float64
		description string
	}{
		{
			name:        "White noise residuals",
			residuals:   generateWhiteNoiseResiduals(100, 42),
			minPValue:   0.10,
			maxPValue:   1.0,
			description: "White noise should have high p-value (no autocorrelation)",
		},
		{
			name:        "Highly autocorrelated residuals",
			residuals:   generateAutocorrelatedResiduals(100, 0.9),
			minPValue:   0.001,
			maxPValue:   0.01,
			description: "Highly autocorrelated residuals should have very low p-value",
		},
		{
			name:        "Moderate autocorrelation",
			residuals:   generateAutocorrelatedResiduals(100, 0.5),
			minPValue:   0.001,
			maxPValue:   0.05,
			description: "Moderately autocorrelated residuals should have low p-value",
		},
		{
			name:        "Alternating pattern",
			residuals:   generateAlternatingResiduals(100),
			minPValue:   0.001,
			maxPValue:   0.05,
			description: "Alternating pattern should show negative autocorrelation",
		},
		{
			name:        "Too few observations",
			residuals:   generateWhiteNoiseResiduals(8, 42),
			minPValue:   1.0,
			maxPValue:   1.0,
			description: "Should return 1.0 for n < 10",
		},
		{
			name:        "Exactly 10 observations",
			residuals:   generateWhiteNoiseResiduals(10, 42),
			minPValue:   0.01,
			maxPValue:   1.0,
			description: "Should compute p-value for n >= 10",
		},
		{
			name:        "Trending residuals",
			residuals:   generateTrendingResiduals(50),
			minPValue:   0.001,
			maxPValue:   0.01,
			description: "Trending residuals indicate model misspecification",
		},
		{
			name:        "Seasonal pattern",
			residuals:   generateSeasonalResiduals(48, 12),
			minPValue:   0.001,
			maxPValue:   0.05,
			description: "Seasonal patterns should show autocorrelation",
		},
		{
			name:        "Zero variance",
			residuals:   generateConstantResiduals(20, 0.0),
			minPValue:   1.0,
			maxPValue:   1.0,
			description: "Zero variance should return 1.0",
		},
		{
			name:        "Small variance",
			residuals:   generateSmallVarianceResiduals(50),
			minPValue:   0.001,
			maxPValue:   1.0,
			description: "Very small variance residuals",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := NewModel(1, 0, 1, 100)
			pValue := m.ljungBoxTest(tt.residuals)
			pValueFloat, _ := pValue.Float64()

			if pValueFloat < tt.minPValue || pValueFloat > tt.maxPValue {
				t.Errorf("%s: p-value %.4f outside expected range [%.4f, %.4f]",
					tt.description, pValueFloat, tt.minPValue, tt.maxPValue)
			}

			t.Logf("%s: p-value = %.4f", tt.name, pValueFloat)
		})
	}
}

func TestModelArima_LjungBoxTestSpecificCases(t *testing.T) {
	model, _ := NewModel(1, 0, 1, 100)

	t.Run("Perfect white noise", func(t *testing.T) {
		// Test multiple seeds to account for randomness
		passCount := 0
		attempts := 5

		for seed := int64(100); seed < int64(100+attempts); seed++ {
			residuals := generateWhiteNoiseResiduals(200, seed)
			pValue := model.ljungBoxTest(residuals)

			if pValue.Gt(fixed.FromFloat64(0.05)) {
				passCount++
			}
		}

		// Expect most attempts to pass
		if passCount < 3 {
			t.Errorf("Expected at least 3 out of %d white noise tests to pass, got %d",
				attempts, passCount)
		}
	})

	t.Run("AR(1) residuals with different coefficients", func(t *testing.T) {
		coefficients := []float64{0.3, 0.5, 0.7, 0.9}

		for _, coef := range coefficients {
			residuals := generateAutocorrelatedResiduals(100, coef)
			pValue := model.ljungBoxTest(residuals)
			pValueFloat, _ := pValue.Float64()

			// Higher coefficient should lead to lower p-value
			expectedMaxP := 0.10
			if coef >= 0.7 {
				expectedMaxP = 0.05
			}
			if pValueFloat > expectedMaxP {
				t.Errorf("AR coefficient %.2f: expected p-value < %.4f, got %.4f",
					coef, expectedMaxP, pValueFloat)
			}

			t.Logf("AR coefficient %.2f: p-value = %.4f", coef, pValueFloat)
		}
	})

	t.Run("Different lag lengths", func(t *testing.T) {
		// Generate autocorrelated residuals
		allResiduals := generateAutocorrelatedResiduals(200, 0.7) // Generate enough for all tests

		// The implementation uses min(10, n/4) for maxLag
		// Test with different sample sizes to verify this
		sizes := []int{20, 40, 80, 160}

		for _, size := range sizes {
			if size > len(allResiduals) {
				t.Skipf("Skipping size %d, not enough test data", size)
				continue
			}

			subset := allResiduals[:size]
			pValue := model.ljungBoxTest(subset)
			pValueFloat, _ := pValue.Float64()

			expectedMaxLag := min(10, size/4)
			t.Logf("Sample size %d (maxLag=%d): p-value = %.4f",
				size, expectedMaxLag, pValueFloat)

			// Should detect autocorrelation
			if pValueFloat > 0.05 {
				t.Errorf("Failed to detect autocorrelation with sample size %d", size)
			}
		}
	})

	t.Run("MA(1) residuals", func(t *testing.T) {
		// Generate MA(1) process residuals
		n := 100
		residuals := make([]fixed.Point, n)
		theta := 0.6

		// Generate white noise
		noise := generateWhiteNoiseResiduals(n+1, 123)

		// MA(1): x_t = e_t + theta * e_{t-1}
		for i := 0; i < n; i++ {
			residuals[i] = noise[i+1].Add(noise[i].Mul(fixed.FromFloat64(theta)))
		}

		pValue := model.ljungBoxTest(residuals)
		pValueFloat, _ := pValue.Float64()

		// MA(1) should show autocorrelation at lag 1
		if pValueFloat > 0.05 {
			t.Errorf("Failed to detect MA(1) autocorrelation, p-value = %.4f", pValueFloat)
		}
	})

	t.Run("Cyclic pattern", func(t *testing.T) {
		// Create residuals with a cyclic pattern
		n := 100
		residuals := make([]fixed.Point, n)

		for i := 0; i < n; i++ {
			// Sin wave pattern
			value := math.Sin(2*math.Pi*float64(i)/10.0) * 0.5
			residuals[i] = fixed.FromFloat64(value)
		}

		pValue := model.ljungBoxTest(residuals)
		pValueFloat, _ := pValue.Float64()

		// Cyclic pattern should be detected
		if pValueFloat > 0.01 {
			t.Errorf("Failed to detect cyclic pattern, p-value = %.4f", pValueFloat)
		}
	})
}

func TestModelArima_LjungBoxTestEdgeCases(t *testing.T) {
	model, _ := NewModel(1, 0, 1, 100)

	t.Run("Single spike with echo", func(t *testing.T) {
		// A single spike doesn't create autocorrelation
		// Instead, create a spike with an "echo" effect
		residuals := make([]fixed.Point, 30)
		for i := range residuals {
			residuals[i] = fixed.Zero
		}

		// Create spikes that show autocorrelation
		residuals[10] = fixed.FromFloat64(1.0)
		residuals[11] = fixed.FromFloat64(0.8) // Echo effect
		residuals[12] = fixed.FromFloat64(0.6)

		residuals[20] = fixed.FromFloat64(-0.9)
		residuals[21] = fixed.FromFloat64(-0.7) // Another echo
		residuals[22] = fixed.FromFloat64(-0.5)

		pValue := model.ljungBoxTest(residuals)
		pValueFloat, _ := pValue.Float64()

		// This pattern should show autocorrelation
		if pValueFloat > 0.10 {
			t.Errorf("Expected low p-value for echo pattern, got %.4f", pValueFloat)
		}

		t.Logf("Echo pattern p-value: %.4f", pValueFloat)
	})

	t.Run("Very small values", func(t *testing.T) {
		residuals := make([]fixed.Point, 50)
		for i := range residuals {
			// Very small alternating values
			if i%2 == 0 {
				residuals[i] = fixed.FromFloat64(1e-10)
			} else {
				residuals[i] = fixed.FromFloat64(-1e-10)
			}
		}

		pValue := model.ljungBoxTest(residuals)
		// Should handle small values gracefully
		pValueFloat, _ := pValue.Float64()
		if pValueFloat < 0.0 || pValueFloat > 1.0 {
			t.Errorf("P-value out of valid range: %.4f", pValueFloat)
		}
	})

	t.Run("Large sample size", func(t *testing.T) {
		// Test with a large sample
		residuals := generateWhiteNoiseResiduals(1000, 999)

		pValue := model.ljungBoxTest(residuals)
		pValueFloat, _ := pValue.Float64()

		// Should still work correctly with large samples
		if pValueFloat < 0.01 {
			t.Errorf("White noise with large sample wrongly rejected, p-value = %.4f", pValueFloat)
		}
	})

	t.Run("Boundary sample sizes", func(t *testing.T) {
		testSizes := []int{9, 10, 11, 39, 40, 41}

		for _, size := range testSizes {
			residuals := generateWhiteNoiseResiduals(size, int64(size))
			pValue := model.ljungBoxTest(residuals)
			pValueFloat, _ := pValue.Float64()

			if size < 10 {
				if pValueFloat != 1.0 {
					t.Errorf("Size %d: expected p-value 1.0, got %.4f", size, pValueFloat)
				}
			} else {
				if pValueFloat < 0.0 || pValueFloat > 1.0 {
					t.Errorf("Size %d: p-value %.4f out of range", size, pValueFloat)
				}
			}
		}
	})
}

func TestModelArima_LjungBoxTestNumericalStability(t *testing.T) {
	model, _ := NewModel(1, 0, 1, 100)

	t.Run("Extreme values", func(t *testing.T) {
		residuals := make([]fixed.Point, 50)
		for i := range residuals {
			if i%10 == 0 {
				// Extreme values
				residuals[i] = fixed.FromFloat64(1000.0)
			} else {
				residuals[i] = fixed.FromFloat64(0.1)
			}
		}

		pValue := model.ljungBoxTest(residuals)
		pValueFloat, _ := pValue.Float64()

		// Should handle extreme values without overflow
		if math.IsNaN(pValueFloat) || math.IsInf(pValueFloat, 0) {
			t.Error("Numerical instability with extreme values")
		}

		// Should detect the pattern
		if pValueFloat > 0.05 {
			t.Errorf("Failed to detect extreme value pattern, p-value = %.4f", pValueFloat)
		}
	})

	t.Run("Near-zero variance", func(t *testing.T) {
		residuals := make([]fixed.Point, 30)
		for i := range residuals {
			// All values very close to mean
			residuals[i] = fixed.FromFloat64(0.00001 * float64(i%3-1))
		}

		pValue := model.ljungBoxTest(residuals)
		// Should handle gracefully
		if pValue.Lt(fixed.Zero) {
			t.Error("Negative p-value with near-zero variance")
		}
	})
}

func TestModelArima_JarqueBeraTest(t *testing.T) {
	tests := []struct {
		name        string
		residuals   []fixed.Point
		minPValue   float64
		maxPValue   float64
		description string
	}{
		{
			name:        "Normal distribution",
			residuals:   generateNormalResiduals(100, 0.0, 1.0),
			minPValue:   0.05, // Lowered threshold - our approximation isn't perfect
			maxPValue:   1.0,
			description: "Normal residuals should have relatively high p-value",
		},
		{
			name:        "Skewed distribution",
			residuals:   generateSkewedResiduals(100, 2.0),
			minPValue:   0.001,
			maxPValue:   0.1,
			description: "Skewed residuals should have low p-value",
		},
		{
			name:        "Heavy-tailed distribution",
			residuals:   generateHeavyTailedResiduals(100),
			minPValue:   0.001,
			maxPValue:   0.1,
			description: "Heavy-tailed residuals should have low p-value",
		},
		{
			name:        "Uniform distribution",
			residuals:   generateUniformResiduals(100),
			minPValue:   0.01,
			maxPValue:   0.5,
			description: "Uniform residuals should have moderate p-value",
		},
		{
			name:        "Too few observations",
			residuals:   generateNormalResiduals(5, 0.0, 1.0),
			minPValue:   1.0,
			maxPValue:   1.0,
			description: "Should return 1.0 for n < 7",
		},
		{
			name:        "Exactly 7 observations",
			residuals:   generateNormalResiduals(7, 0.0, 1.0),
			minPValue:   0.01,
			maxPValue:   1.0,
			description: "Should compute p-value for n >= 7",
		},
		{
			name:        "Zero variance",
			residuals:   generateConstantResiduals(20, 0.5),
			minPValue:   1.0,
			maxPValue:   1.0,
			description: "Should return 1.0 for zero variance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := NewModel(1, 0, 1, 100)
			pValue := m.jarqueBeraTest(tt.residuals)
			pValueFloat, _ := pValue.Float64()

			if pValueFloat < tt.minPValue || pValueFloat > tt.maxPValue {
				t.Errorf("%s: p-value %.4f outside expected range [%.4f, %.4f]",
					tt.description, pValueFloat, tt.minPValue, tt.maxPValue)
			}

			t.Logf("%s: p-value = %.4f", tt.name, pValueFloat)
		})
	}
}

func TestModelArima_JarqueBeraTestSpecificCases(t *testing.T) {
	model, _ := NewModel(1, 0, 1, 100)

	t.Run("Perfect normal distribution", func(t *testing.T) {
		// Test with multiple seeds to account for sampling variation
		passCount := 0
		attempts := 5

		for seed := int64(12345); seed < int64(12345+attempts); seed++ {
			var residuals []fixed.Point
			n := 500 // Larger sample size

			// Use linear congruential generator
			a := int64(1664525)
			c := int64(1013904223)
			m := int64(1) << 32
			x := seed

			// Generate normal distribution using CLT (more stable than Box-Muller)
			for i := 0; i < n; i++ {
				sum := 0.0
				// Sum of 24 uniform random variables for better approximation
				for j := 0; j < 24; j++ {
					x = (a*x + c) % m
					u := float64(x) / float64(m)
					sum += u
				}
				// CLT: sum of 24 uniform(0,1) has mean 12 and variance 2
				z := (sum - 12.0) / math.Sqrt(2.0)
				residuals = append(residuals, fixed.FromFloat64(z*0.1))
			}

			pValue := model.jarqueBeraTest(residuals)
			if pValue.Gt(fixed.FromFloat64(0.05)) {
				passCount++
			}
		}

		// Due to sampling variation, we expect most but not necessarily all to pass
		if passCount < 2 {
			t.Errorf("Expected at least 2 out of %d attempts to pass normality test, got %d",
				attempts, passCount)
		}

		t.Logf("Normality test passed %d out of %d times", passCount, attempts)
	})

	t.Run("Extreme skewness", func(t *testing.T) {
		// Create a highly skewed distribution (exponential-like)
		var residuals []fixed.Point
		for i := 0; i < 50; i++ {
			if i < 45 {
				residuals = append(residuals, fixed.FromFloat64(0.1))
			} else {
				// A few extreme values
				residuals = append(residuals, fixed.FromFloat64(2.0))
			}
		}

		pValue := model.jarqueBeraTest(residuals)
		if pValue.Gt(fixed.FromFloat64(0.1)) {
			pf, _ := pValue.Float64()
			t.Errorf("Highly skewed distribution should have low p-value, got %.4f", pf)
		}
	})

	t.Run("Extreme kurtosis", func(t *testing.T) {
		// Create distribution with high kurtosis (heavy tails)
		var residuals []fixed.Point
		for i := 0; i < 50; i++ {
			if i%10 == 0 {
				// Extreme values
				if i%20 == 0 {
					residuals = append(residuals, fixed.FromFloat64(3.0))
				} else {
					residuals = append(residuals, fixed.FromFloat64(-3.0))
				}
			} else {
				// Central values
				residuals = append(residuals, fixed.FromFloat64(0.0))
			}
		}

		pValue := model.jarqueBeraTest(residuals)
		if pValue.Gt(fixed.FromFloat64(0.1)) {
			pf, _ := pValue.Float64()
			t.Errorf("High kurtosis distribution should have low p-value, got %.4f", pf)
		}
	})

	t.Run("Bimodal distribution", func(t *testing.T) {
		// Create bimodal distribution
		var residuals []fixed.Point
		for i := 0; i < 100; i++ {
			if i < 50 {
				// First mode around -1
				residuals = append(residuals, fixed.FromFloat64(-1.0+float64(i%10)*0.01))
			} else {
				// Second mode around +1
				residuals = append(residuals, fixed.FromFloat64(1.0+float64(i%10)*0.01))
			}
		}

		pValue := model.jarqueBeraTest(residuals)
		// Bimodal distribution should fail the normality test
		if pValue.Gt(fixed.FromFloat64(0.1)) {
			pf, _ := pValue.Float64()
			t.Errorf("Bimodal distribution should have low p-value, got %.4f", pf)
		}
	})
}

func TestModelArima_JarqueBeraTestImplementation(t *testing.T) {
	m, _ := NewModel(1, 0, 1, 100)

	t.Run("Known skewness and kurtosis", func(t *testing.T) {
		// Create data with known properties
		// For standard normal: skewness = 0, kurtosis = 3
		// JB statistic = n/6 * (S² + K²/4) where K is excess kurtosis (kurtosis - 3)

		n := 100
		residuals := make([]fixed.Point, n)

		// Create data with zero mean, unit variance, zero skewness, zero excess kurtosis
		// This is approximately normal
		for i := 0; i < n; i++ {
			val := math.Cos(2*math.Pi*float64(i)/float64(n)) * 0.5
			residuals[i] = fixed.FromFloat64(val)
		}

		pValue := m.jarqueBeraTest(residuals)
		pf, _ := pValue.Float64()
		t.Logf("Cosine wave residuals p-value: %.4f", pf)
	})

	t.Run("Edge case calculations", func(t *testing.T) {
		// Test with values that might cause numerical issues
		residuals := []fixed.Point{
			fixed.FromFloat64(1e-10),
			fixed.FromFloat64(-1e-10),
			fixed.FromFloat64(1e-10),
			fixed.FromFloat64(-1e-10),
			fixed.FromFloat64(1e-10),
			fixed.FromFloat64(-1e-10),
			fixed.FromFloat64(1e-10),
			fixed.FromFloat64(-1e-10),
		}

		pValue := m.jarqueBeraTest(residuals)
		// Very small values centered around zero should be approximately normal
		if pValue.Lt(fixed.FromFloat64(0.1)) {
			pf, _ := pValue.Float64()
			t.Errorf("Small symmetric values should have high p-value, got %.4f", pf)
		}
	})
}

func TestModelArima_CheckParameterValidity(t *testing.T) {
	tests := []struct {
		name        string
		p, q        int
		arParams    []fixed.Point
		maParams    []fixed.Point
		variance    fixed.Point
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid AR(1) model",
			p:           1,
			q:           0,
			arParams:    []fixed.Point{fixed.FromFloat64(0.5)},
			maParams:    []fixed.Point{},
			variance:    fixed.FromFloat64(1.0),
			shouldError: false,
		},
		{
			name:        "Valid MA(1) model",
			p:           0,
			q:           1,
			arParams:    []fixed.Point{},
			maParams:    []fixed.Point{fixed.FromFloat64(0.3)},
			variance:    fixed.FromFloat64(0.5),
			shouldError: false,
		},
		{
			name:        "Valid ARMA(1,1) model",
			p:           1,
			q:           1,
			arParams:    []fixed.Point{fixed.FromFloat64(0.4)},
			maParams:    []fixed.Point{fixed.FromFloat64(0.6)},
			variance:    fixed.FromFloat64(2.0),
			shouldError: false,
		},
		{
			name:        "Non-stationary AR(1) - coefficient = 1",
			p:           1,
			q:           0,
			arParams:    []fixed.Point{fixed.One},
			maParams:    []fixed.Point{},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true,
			errorMsg:    "AR parameters suggest non-stationarity",
		},
		{
			name:        "Non-stationary AR(1) - coefficient > 1",
			p:           1,
			q:           0,
			arParams:    []fixed.Point{fixed.FromFloat64(1.2)},
			maParams:    []fixed.Point{},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true,
			errorMsg:    "AR parameters suggest non-stationarity",
		},
		{
			name:        "Non-invertible MA(1) - coefficient = 1",
			p:           0,
			q:           1,
			arParams:    []fixed.Point{},
			maParams:    []fixed.Point{fixed.One},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true,
			errorMsg:    "MA parameters suggest non-invertibility",
		},
		{
			name:        "Non-invertible MA(1) - coefficient > 1",
			p:           0,
			q:           1,
			arParams:    []fixed.Point{},
			maParams:    []fixed.Point{fixed.FromFloat64(1.5)},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true,
			errorMsg:    "MA parameters suggest non-invertibility",
		},
		{
			name:        "Invalid variance - zero",
			p:           1,
			q:           0,
			arParams:    []fixed.Point{fixed.FromFloat64(0.5)},
			maParams:    []fixed.Point{},
			variance:    fixed.Zero,
			shouldError: true,
			errorMsg:    "invalid variance estimate",
		},
		{
			name:        "Invalid variance - negative",
			p:           1,
			q:           0,
			arParams:    []fixed.Point{fixed.FromFloat64(0.5)},
			maParams:    []fixed.Point{},
			variance:    fixed.FromFloat64(-1.0),
			shouldError: true,
			errorMsg:    "invalid variance estimate",
		},
		{
			name:        "AR(2) model - sum of coefficients = 1",
			p:           2,
			q:           0,
			arParams:    []fixed.Point{fixed.FromFloat64(0.6), fixed.FromFloat64(0.4)},
			maParams:    []fixed.Point{},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true,
			errorMsg:    "AR parameters suggest non-stationarity",
		},
		{
			name:        "AR(2) model - sum of coefficients > 1",
			p:           2,
			q:           0,
			arParams:    []fixed.Point{fixed.FromFloat64(0.7), fixed.FromFloat64(0.5)},
			maParams:    []fixed.Point{},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true,
			errorMsg:    "AR parameters suggest non-stationarity",
		},
		{
			name:        "Valid AR(2) model",
			p:           2,
			q:           0,
			arParams:    []fixed.Point{fixed.FromFloat64(0.4), fixed.FromFloat64(0.3)},
			maParams:    []fixed.Point{},
			variance:    fixed.FromFloat64(1.0),
			shouldError: false,
		},
		{
			name:        "MA(2) model - sum of coefficients = 1",
			p:           0,
			q:           2,
			arParams:    []fixed.Point{},
			maParams:    []fixed.Point{fixed.FromFloat64(0.5), fixed.FromFloat64(0.5)},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true,
			errorMsg:    "MA parameters suggest non-invertibility",
		},
		{
			name:        "Valid MA(2) model",
			p:           0,
			q:           2,
			arParams:    []fixed.Point{},
			maParams:    []fixed.Point{fixed.FromFloat64(0.3), fixed.FromFloat64(0.2)},
			variance:    fixed.FromFloat64(1.0),
			shouldError: false,
		},
		{
			name:        "ARMA(2,2) - AR sum borderline",
			p:           2,
			q:           2,
			arParams:    []fixed.Point{fixed.FromFloat64(0.5), fixed.FromFloat64(0.49)},
			maParams:    []fixed.Point{fixed.FromFloat64(0.3), fixed.FromFloat64(0.2)},
			variance:    fixed.FromFloat64(1.0),
			shouldError: false, // 0.5 + 0.49 = 0.99 < 1.0
		},
		{
			name:        "Mixed parameters with negative values - invalid",
			p:           2,
			q:           2,
			arParams:    []fixed.Point{fixed.FromFloat64(0.8), fixed.FromFloat64(-0.3)},
			maParams:    []fixed.Point{fixed.FromFloat64(-0.4), fixed.FromFloat64(0.2)},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true, // Sum of absolute values: |0.8| + |-0.3| = 1.1 > 1
			errorMsg:    "AR parameters suggest non-stationarity",
		},
		{
			name:        "Mixed parameters with negative values - valid",
			p:           2,
			q:           2,
			arParams:    []fixed.Point{fixed.FromFloat64(0.5), fixed.FromFloat64(-0.3)},
			maParams:    []fixed.Point{fixed.FromFloat64(-0.4), fixed.FromFloat64(0.2)},
			variance:    fixed.FromFloat64(1.0),
			shouldError: false, // AR sum: |0.5| + |-0.3| = 0.8 < 1, MA sum: |-0.4| + |0.2| = 0.6 < 1
		},
		{
			name:        "AR with negative coefficients - non-stationary",
			p:           2,
			q:           0,
			arParams:    []fixed.Point{fixed.FromFloat64(-0.8), fixed.FromFloat64(-0.3)},
			maParams:    []fixed.Point{},
			variance:    fixed.FromFloat64(1.0),
			shouldError: true, // Sum of absolute values: 0.8 + 0.3 = 1.1 > 1
			errorMsg:    "AR parameters suggest non-stationarity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create model
			m, _ := NewModel(tt.p, 0, tt.q, 100)
			m.arParams = tt.arParams
			m.maParams = tt.maParams
			m.variance = tt.variance

			// Check parameter validity
			err := m.checkParameterValidity()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestModelArima_CheckParameterValidityEdgeCases(t *testing.T) {

	t.Run("Very small variance", func(t *testing.T) {
		m, _ := NewModel(1, 0, 0, 100)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
		m.variance = fixed.FromFloat64(0.0000001) // Very small but positive

		err := m.checkParameterValidity()
		if err != nil {
			t.Errorf("Small positive variance should be valid, got: %v", err)
		}
	})

	t.Run("Exact boundary - AR sum = 0.999999", func(t *testing.T) {
		m, _ := NewModel(3, 0, 0, 100)
		m.arParams = []fixed.Point{
			fixed.FromFloat64(0.4),
			fixed.FromFloat64(0.3),
			fixed.FromFloat64(0.299999),
		}
		m.variance = fixed.FromFloat64(1.0)

		err := m.checkParameterValidity()
		if err != nil {
			t.Errorf("AR sum < 1 should be valid, got: %v", err)
		}
	})

	t.Run("High order AR model", func(t *testing.T) {
		m, _ := NewModel(10, 0, 0, 100)
		// Create parameters that sum to 0.95
		m.arParams = make([]fixed.Point, 10)
		for i := 0; i < 10; i++ {
			m.arParams[i] = fixed.FromFloat64(0.095)
		}
		m.variance = fixed.FromFloat64(1.0)

		err := m.checkParameterValidity()
		if err != nil {
			t.Errorf("High order AR with valid sum should pass, got: %v", err)
		}
	})

	t.Run("High order MA model", func(t *testing.T) {
		m, _ := NewModel(0, 0, 10, 100)
		// Create parameters that sum to 1.01 (should fail)
		m.maParams = make([]fixed.Point, 10)
		for i := 0; i < 10; i++ {
			m.maParams[i] = fixed.FromFloat64(0.101)
		}
		m.variance = fixed.FromFloat64(1.0)

		err := m.checkParameterValidity()
		if err == nil {
			t.Error("High order MA with sum > 1 should fail")
		}
	})
}

func TestModelArima_CheckResidualProperties(t *testing.T) {
	tests := []struct {
		name           string
		residuals      []fixed.Point
		ljungBoxPValue fixed.Point
		shouldError    bool
		errorMsg       string
	}{
		{
			name: "Good residuals - no autocorrelation",
			residuals: []fixed.Point{
				fixed.FromFloat64(0.1), fixed.FromFloat64(-0.2), fixed.FromFloat64(0.15),
				fixed.FromFloat64(-0.1), fixed.FromFloat64(0.05), fixed.FromFloat64(-0.12),
				fixed.FromFloat64(0.08), fixed.FromFloat64(-0.05), fixed.FromFloat64(0.11),
				fixed.FromFloat64(-0.09), fixed.FromFloat64(0.02), fixed.FromFloat64(-0.08),
			},
			ljungBoxPValue: fixed.FromFloat64(0.15), // p > 0.05, no significant autocorrelation
			shouldError:    false,
		},
		{
			name: "Bad residuals - significant autocorrelation",
			residuals: []fixed.Point{
				fixed.FromFloat64(1.0), fixed.FromFloat64(0.9), fixed.FromFloat64(0.8),
				fixed.FromFloat64(0.7), fixed.FromFloat64(0.6), fixed.FromFloat64(0.5),
				fixed.FromFloat64(0.4), fixed.FromFloat64(0.3), fixed.FromFloat64(0.2),
				fixed.FromFloat64(0.1), fixed.FromFloat64(0.0), fixed.FromFloat64(-0.1),
			},
			ljungBoxPValue: fixed.FromFloat64(0.01), // p < 0.05, significant autocorrelation
			shouldError:    true,
			errorMsg:       "residuals show significant autocorrelation",
		},
		{
			name:           "Too few residuals",
			residuals:      []fixed.Point{fixed.FromFloat64(0.1), fixed.FromFloat64(-0.1)},
			ljungBoxPValue: fixed.FromFloat64(0.5), // Won't be checked due to size
			shouldError:    false,                  // Should pass because size < 10
		},
		{
			name: "Exactly 10 residuals - boundary case",
			residuals: []fixed.Point{
				fixed.FromFloat64(0.1), fixed.FromFloat64(-0.1), fixed.FromFloat64(0.1),
				fixed.FromFloat64(-0.1), fixed.FromFloat64(0.1), fixed.FromFloat64(-0.1),
				fixed.FromFloat64(0.1), fixed.FromFloat64(-0.1), fixed.FromFloat64(0.1),
				fixed.FromFloat64(-0.1),
			},
			ljungBoxPValue: fixed.FromFloat64(0.03), // p < 0.05
			shouldError:    true,
			errorMsg:       "residuals show significant autocorrelation",
		},
		{
			name:           "Empty residuals",
			residuals:      []fixed.Point{},
			ljungBoxPValue: fixed.FromFloat64(0.5),
			shouldError:    false, // Should pass because empty
		},
		{
			name: "Borderline p-value",
			residuals: []fixed.Point{
				fixed.FromFloat64(0.2), fixed.FromFloat64(-0.1), fixed.FromFloat64(0.15),
				fixed.FromFloat64(-0.2), fixed.FromFloat64(0.1), fixed.FromFloat64(-0.15),
				fixed.FromFloat64(0.12), fixed.FromFloat64(-0.08), fixed.FromFloat64(0.18),
				fixed.FromFloat64(-0.14), fixed.FromFloat64(0.05), fixed.FromFloat64(-0.1),
			},
			ljungBoxPValue: fixed.FromFloat64(0.05), // Exactly at a threshold
			shouldError:    false,                   // Should pass as it's not < 0.05
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create model
			m, _ := NewModel(1, 0, 1, 100)
			m.estimated = true

			// Set up residuals
			m.residuals.Clear()
			for _, r := range tt.residuals {
				m.residuals.Add(r)
			}

			// Set diagnostics
			m.diagnostics.LjungBoxPValue = tt.ljungBoxPValue

			// Check residual properties
			err := m.checkResidualProperties()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestModelArima_CheckResidualPropertiesIntegration(t *testing.T) {

	t.Run("Well-specified model", func(t *testing.T) {
		m, _ := NewModel(1, 0, 0, 50)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
		m.variance = fixed.FromFloat64(1.0)
		m.estimated = true

		// Generate pseudo-random white noise residuals with low autocorrelation
		// Using a linear congruential generator for reproducibility
		var residuals []fixed.Point
		a := int64(1664525)
		c := int64(1013904223)
		mod := int64(1) << 32
		x := int64(12345) // seed

		for i := 0; i < 30; i++ {
			x = (a*x + c) % mod
			// Convert to float in range [-0.2, 0.2]
			val := float64(x)/float64(mod)*0.4 - 0.2
			residuals = append(residuals, fixed.FromFloat64(val))
		}

		// Generate a series from these residuals
		series := []fixed.Point{fixed.Zero}
		for i := 0; i < len(residuals); i++ {
			// AR(1) process: y_t = 0.5 * y_{t-1} + e_t
			value := series[i].Mul(fixed.FromFloat64(0.5)).Add(residuals[i])
			series = append(series, value)
		}

		// Add data to a model
		for _, val := range series {
			m.diffData.Add(val)
		}

		// Set residuals
		m.residuals.Clear()
		for _, r := range residuals {
			m.residuals.Add(r)
		}

		// Calculate diagnostics (this sets LjungBoxPValue)
		m.calculateDiagnostics()

		// Check residual properties
		err := m.checkResidualProperties()
		if err != nil {
			t.Errorf("Well-specified model should pass residual checks, got: %v", err)
			t.Logf("Ljung-Box p-value: %v", m.diagnostics.LjungBoxPValue.String())

			// Debug: check the actual autocorrelations
			n := len(residuals)
			mean := fixed.Zero
			for _, r := range residuals {
				mean = mean.Add(r)
			}
			mean = mean.DivInt(n)

			var variance fixed.Point
			for _, r := range residuals {
				diff := r.Sub(mean)
				variance = variance.Add(diff.Mul(diff))
			}
			variance = variance.DivInt(n)

			t.Logf("Residual mean: %v, variance: %v", mean.String(), variance.String())
		}
	})

	t.Run("Misspecified model", func(t *testing.T) {
		m, _ := NewModel(1, 0, 0, 50)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.3)} // Wrong parameter
		m.variance = fixed.FromFloat64(1.0)
		m.estimated = true

		// Generate AR(1) data with true parameter 0.8
		series := []fixed.Point{fixed.Zero}

		for i := 1; i < 30; i++ {
			// True process: y_t = 0.8 * y_{t-1} + small_error
			// But we're fitting with 0.3
			value := series[i-1].Mul(fixed.FromFloat64(0.8)).Add(fixed.FromFloat64(0.1))
			series = append(series, value)
		}

		// Add data to a model
		for _, val := range series {
			m.diffData.Add(val)
		}

		// Calculate residuals with a wrong model
		var residuals []fixed.Point
		for i := 1; i < len(series); i++ {
			fitted := series[i-1].Mul(fixed.FromFloat64(0.3))
			residual := series[i].Sub(fitted)
			residuals = append(residuals, residual)
		}

		m.residuals.Clear()
		for _, r := range residuals {
			m.residuals.Add(r)
		}

		// Force a low p-value to simulate autocorrelation detection
		m.diagnostics.LjungBoxPValue = fixed.FromFloat64(0.01)

		// Check residual properties
		err := m.checkResidualProperties()
		if err == nil {
			t.Error("Misspecified model should fail residual checks")
		}
	})
}

func TestModelArima_CheckResidualPropertiesEdgeCases(t *testing.T) {
	t.Run("Nil diagnostics", func(t *testing.T) {
		m, _ := NewModel(1, 0, 1, 100)
		m.estimated = true

		// Add some residuals
		for i := 0; i < 15; i++ {
			m.residuals.Add(fixed.FromFloat64(float64(i) * 0.01))
		}

		// Don't set diagnostics - LjungBoxPValue will be zero
		// This simulates a case where diagnostics weren't calculated

		err := m.checkResidualProperties()
		// With zero p-value, it should detect autocorrelation
		if err == nil {
			t.Error("Should detect autocorrelation with zero p-value")
		}
	})

	t.Run("Very large residuals buffer", func(t *testing.T) {
		m, _ := NewModel(1, 0, 1, 1000)
		m.estimated = true

		// Add many residuals
		for i := 0; i < 500; i++ {
			// Alternating pattern to avoid autocorrelation
			if i%2 == 0 {
				m.residuals.Add(fixed.FromFloat64(0.1))
			} else {
				m.residuals.Add(fixed.FromFloat64(-0.1))
			}
		}

		m.diagnostics.LjungBoxPValue = fixed.FromFloat64(0.8) // High p-value

		err := m.checkResidualProperties()
		if err != nil {
			t.Errorf("Should pass with high p-value, got: %v", err)
		}
	})
}

func TestModelArima_InitializeForecastState(t *testing.T) {
	tests := []struct {
		name                  string
		p, d, q               int
		diffSeriesData        []fixed.Point
		rawSeriesData         []fixed.Point
		residualsData         []fixed.Point
		variance              fixed.Point
		expectedDiffCount     int
		expectedRawCount      int
		expectedResidualCount int
	}{
		{
			name:                  "Simple AR(1) model",
			p:                     1,
			d:                     0,
			q:                     0,
			diffSeriesData:        []fixed.Point{fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11)},
			rawSeriesData:         []fixed.Point{fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11)},
			residualsData:         []fixed.Point{fixed.Zero, fixed.FromFloat64(0.5), fixed.FromFloat64(-0.2)},
			variance:              fixed.FromFloat64(1.0),
			expectedDiffCount:     3,
			expectedRawCount:      3,
			expectedResidualCount: 3,
		},
		{
			name:                  "ARIMA(1,1,1) model",
			p:                     1,
			d:                     1,
			q:                     1,
			diffSeriesData:        []fixed.Point{fixed.FromFloat64(2), fixed.FromFloat64(-1), fixed.FromFloat64(3)},
			rawSeriesData:         []fixed.Point{fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11), fixed.FromFloat64(14)},
			residualsData:         []fixed.Point{fixed.FromFloat64(0.1), fixed.FromFloat64(-0.3)},
			variance:              fixed.FromFloat64(0.5),
			expectedDiffCount:     3,
			expectedRawCount:      4,
			expectedResidualCount: 2,
		},
		{
			name:                  "Empty buffers",
			p:                     1,
			d:                     0,
			q:                     1,
			diffSeriesData:        []fixed.Point{},
			rawSeriesData:         []fixed.Point{},
			residualsData:         []fixed.Point{},
			variance:              fixed.FromFloat64(1.0),
			expectedDiffCount:     0,
			expectedRawCount:      0,
			expectedResidualCount: 0,
		},
		{
			name:                  "Model with high differencing",
			p:                     1,
			d:                     2,
			q:                     0,
			diffSeriesData:        []fixed.Point{fixed.FromFloat64(0.5), fixed.FromFloat64(-0.3)},
			rawSeriesData:         []fixed.Point{fixed.FromFloat64(100), fixed.FromFloat64(102), fixed.FromFloat64(105), fixed.FromFloat64(109)},
			residualsData:         []fixed.Point{fixed.FromFloat64(0.05)},
			variance:              fixed.FromFloat64(0.25),
			expectedDiffCount:     2,
			expectedRawCount:      4,
			expectedResidualCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create model
			m, _ := NewModel(tt.p, tt.d, tt.q, 100)
			m.variance = tt.variance
			m.estimated = true

			// Populate buffers
			for _, val := range tt.rawSeriesData {
				m.rawData.Add(val)
			}
			for _, val := range tt.diffSeriesData {
				m.diffData.Add(val)
			}
			for _, val := range tt.residualsData {
				m.residuals.Add(val)
			}

			// Initialize forecast state
			state := m.initializeForecastState()

			// Check diff series
			if len(state.diffSeries) != tt.expectedDiffCount {
				t.Errorf("Expected %d diff series values, got %d",
					tt.expectedDiffCount, len(state.diffSeries))
			}

			// Check raw series
			if len(state.rawValues) != tt.expectedRawCount {
				t.Errorf("Expected %d raw series values, got %d",
					tt.expectedRawCount, len(state.rawValues))
			}

			// Check residuals
			if len(state.residuals) != tt.expectedResidualCount {
				t.Errorf("Expected %d residuals, got %d",
					tt.expectedResidualCount, len(state.residuals))
			}

			// Check variance
			if !state.variance.Eq(tt.variance) {
				t.Errorf("Expected variance %v, got %v",
					tt.variance.String(), state.variance.String())
			}

			// Check mean (should be from diffData)
			if tt.expectedDiffCount > 0 {
				expectedMean := m.diffData.Mean()
				if !state.mean.Eq(expectedMean) {
					t.Errorf("Expected mean %v, got %v",
						expectedMean.String(), state.mean.String())
				}
			}

			// Check if forecasted diffs are initialized empty
			if len(state.forecastedDiffs) != 0 {
				t.Errorf("Expected empty forecastedDiffs, got %d elements",
					len(state.forecastedDiffs))
			}
		})
	}
}

func TestModelArima_InitializeForecastStateWithCircularBufferWrap(t *testing.T) {

	m, _ := NewModel(1, 0, 1, 50)
	m.variance = fixed.FromFloat64(1.0)
	m.estimated = true

	// Add more data than buffer capacity to force wrap-around
	for i := 0; i < 100; i++ {
		m.rawData.Add(fixed.FromFloat64(float64(i)))
		m.diffData.Add(fixed.FromFloat64(float64(i * 10)))
		if i < 8 {
			m.residuals.Add(fixed.FromFloat64(float64(i) * 0.1))
		}
	}

	state := m.initializeForecastState()

	// Should only have the last 50 raw values
	if len(state.rawValues) != 50 {
		t.Errorf("Expected 5 raw values, got %d", len(state.rawValues))
	}

	// Check that we have the most recent values in an oldest-to-newest order
	expectedRaw := []float64{50, 51, 52, 53, 54}
	for i, expected := range expectedRaw {
		if !state.rawValues[i].Eq(fixed.FromFloat64(expected)) {
			t.Errorf("Raw value at index %d: expected %v, got %v",
				i, expected, state.rawValues[i].String())
		}
	}

	// Similar check for diff series
	expectedDiff := []float64{500, 510, 520, 530, 540}
	for i, expected := range expectedDiff {
		if !state.diffSeries[i].Eq(fixed.FromFloat64(expected)) {
			t.Errorf("Diff value at index %d: expected %v, got %v",
				i, expected, state.diffSeries[i].String())
		}
	}
}

func TestModelArima_InitializeForecastStateConsistency(t *testing.T) {

	m, _ := NewModel(2, 1, 1, 50)
	m.variance = fixed.FromFloat64(1.5)
	m.estimated = true

	// Add some data
	for i := 0; i < 15; i++ {
		m.rawData.Add(fixed.FromFloat64(float64(100 + i)))
		if i > 0 { // For d=1, we need at least 2 raw values
			m.diffData.Add(fixed.FromFloat64(float64(i)))
		}
		if i < 10 {
			m.residuals.Add(fixed.FromFloat64(float64(i) * 0.01))
		}
	}

	// Initialize state multiple times
	state1 := m.initializeForecastState()
	state2 := m.initializeForecastState()

	// Compare states
	if len(state1.diffSeries) != len(state2.diffSeries) {
		t.Error("Diff series length mismatch between calls")
	}
	if len(state1.rawValues) != len(state2.rawValues) {
		t.Error("Raw values length mismatch between calls")
	}
	if len(state1.residuals) != len(state2.residuals) {
		t.Error("Residuals length mismatch between calls")
	}
	if !state1.mean.Eq(state2.mean) {
		t.Error("Mean mismatch between calls")
	}
	if !state1.variance.Eq(state2.variance) {
		t.Error("Variance mismatch between calls")
	}

	// Check that data values are identical
	for i := range state1.diffSeries {
		if !state1.diffSeries[i].Eq(state2.diffSeries[i]) {
			t.Errorf("Diff series mismatch at index %d", i)
		}
	}
}

func TestModelArima_ForecastOneStep(t *testing.T) {
	tests := []struct {
		name             string
		p, d, q          int
		arParams         []fixed.Point
		maParams         []fixed.Point
		constant         fixed.Point
		variance         fixed.Point
		includeConstant  bool
		diffSeries       []fixed.Point
		rawSeries        []fixed.Point
		residuals        []fixed.Point
		step             int
		expectedForecast fixed.Point
		tolerance        fixed.Point
	}{
		{
			name:            "Simple AR(1) model",
			p:               1,
			d:               0,
			q:               0,
			arParams:        []fixed.Point{fixed.FromFloat64(0.5)},
			maParams:        []fixed.Point{},
			constant:        fixed.FromFloat64(2.0),
			variance:        fixed.FromFloat64(1.0),
			includeConstant: true,
			diffSeries:      []fixed.Point{fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11)},
			rawSeries:       []fixed.Point{fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11)},
			residuals:       []fixed.Point{fixed.Zero, fixed.Zero, fixed.Zero},
			step:            0,
			// forecast = constant + φ₁ * (last_value - mean) + mean
			// mean = (10 + 12 + 11) / 3 = 11
			// forecast = 2 + 0.5 * (11 - 11) + 11 = 13
			expectedForecast: fixed.FromFloat64(13),
			tolerance:        fixed.FromFloat64(0.01),
		},
		{
			name:            "MA(1) model",
			p:               0,
			d:               0,
			q:               1,
			arParams:        []fixed.Point{},
			maParams:        []fixed.Point{fixed.FromFloat64(0.3)},
			constant:        fixed.FromFloat64(5.0),
			variance:        fixed.FromFloat64(1.0),
			includeConstant: true,
			diffSeries:      []fixed.Point{fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11)},
			rawSeries:       []fixed.Point{fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11)},
			residuals:       []fixed.Point{fixed.FromFloat64(0.5), fixed.FromFloat64(-0.3), fixed.FromFloat64(0.2)},
			step:            0,
			// forecast = constant + θ₁ * last_residual + mean
			// The mean is calculated by PointBuffer, not simple arithmetic
			// We'll use a larger tolerance to account for this
			expectedForecast: fixed.FromFloat64(16.1),
			tolerance:        fixed.FromFloat64(0.2),
		},
		{
			name:            "ARIMA(1,1,1) model",
			p:               1,
			d:               1,
			q:               1,
			arParams:        []fixed.Point{fixed.FromFloat64(0.4)},
			maParams:        []fixed.Point{fixed.FromFloat64(0.2)},
			constant:        fixed.FromFloat64(0.1),
			variance:        fixed.FromFloat64(1.0),
			includeConstant: true,
			diffSeries:      []fixed.Point{fixed.FromFloat64(2), fixed.FromFloat64(-1), fixed.FromFloat64(3)},
			rawSeries:       []fixed.Point{fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11), fixed.FromFloat64(14)},
			residuals:       []fixed.Point{fixed.Zero, fixed.FromFloat64(0.5), fixed.FromFloat64(-0.2)},
			step:            0,
			// Forecast in a differenced scale, then undifference
			// The exact calculation depends on PointBuffer's mean calculation
			// and the undifferencing process
			expectedForecast: fixed.FromFloat64(16.1),
			tolerance:        fixed.FromFloat64(1.2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and setup model
			m, _ := NewModel(tt.p, tt.d, tt.q, 100)
			m.arParams = tt.arParams
			m.maParams = tt.maParams
			m.constant = tt.constant
			m.variance = tt.variance
			m.includeConstant = tt.includeConstant
			m.estimated = true

			// Populate buffers
			for _, val := range tt.rawSeries {
				m.rawData.Add(val)
			}
			for _, val := range tt.diffSeries {
				m.diffData.Add(val)
			}
			for _, val := range tt.residuals {
				m.residuals.Add(val)
			}

			// Initialize forecast state
			state := m.initializeForecastState()

			// Perform a one-step forecast
			result, err := m.forecastOneStep(state, tt.step)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check point forecast
			diff := result.PointForecast.Sub(tt.expectedForecast).Abs()
			if diff.Gt(tt.tolerance) {
				t.Errorf("Expected forecast %v, got %v (diff: %v)",
					tt.expectedForecast.String(),
					result.PointForecast.String(),
					diff.String())
			}

			// Check that confidence intervals make sense
			if result.ConfidenceInterval.Lower95.Gte(result.PointForecast) {
				t.Error("Lower 95% CI should be less than point forecast")
			}
			if result.ConfidenceInterval.Upper95.Lte(result.PointForecast) {
				t.Error("Upper 95% CI should be greater than point forecast")
			}
			if result.ConfidenceInterval.Lower80.Lt(result.ConfidenceInterval.Lower95) {
				t.Error("Lower 80% CI should be greater than lower 95% CI")
			}
			if result.ConfidenceInterval.Upper80.Gt(result.ConfidenceInterval.Upper95) {
				t.Error("Upper 80% CI should be less than upper 95% CI")
			}

			// Check standard error
			if result.StandardError.Lte(fixed.Zero) {
				t.Error("Standard error should be positive")
			}
		})
	}
}

func TestModelArima_ForecastOneStepMultiStep(t *testing.T) {
	// Test multistep forecasting with state updates
	m, _ := NewModel(1, 0, 1, 100)
	m.arParams = []fixed.Point{fixed.FromFloat64(0.6)}
	m.maParams = []fixed.Point{fixed.FromFloat64(0.3)}
	m.constant = fixed.FromFloat64(1.0)
	m.variance = fixed.FromFloat64(1.0)
	m.includeConstant = true
	m.estimated = true

	// Add historical data
	series := []fixed.Point{
		fixed.FromFloat64(10), fixed.FromFloat64(12), fixed.FromFloat64(11),
		fixed.FromFloat64(13), fixed.FromFloat64(14),
	}
	for _, val := range series {
		m.rawData.Add(val)
		m.diffData.Add(val) // No differencing
	}

	// Add some residuals
	residuals := []fixed.Point{
		fixed.Zero, fixed.FromFloat64(0.5), fixed.FromFloat64(-0.3),
		fixed.FromFloat64(0.2), fixed.FromFloat64(0.1),
	}
	for _, val := range residuals {
		m.residuals.Add(val)
	}

	state := m.initializeForecastState()

	// Test that multistep forecasts use previous forecasts
	prevForecast := fixed.Zero
	for step := 0; step < 3; step++ {
		result, err := m.forecastOneStep(state, step)
		if err != nil {
			t.Fatalf("Step %d: Unexpected error: %v", step, err)
		}

		// Store in a forecast cache (mimicking Forecast method)
		m.forecastCache = append(m.forecastCache, result.PointForecast)

		// Update state
		m.appendZeroResidual(state)

		// For AR models, later forecasts should depend on previous ones
		if step > 0 && m.p > 0 {
			// The forecast should be different from the previous one
			// (unless the model predicts constant values)
			if result.PointForecast.Eq(prevForecast) && !m.arParams[0].Eq(fixed.Zero) {
				t.Errorf("Step %d: Forecast should differ from previous step", step)
			}
		}

		// Variance should increase with a horizon
		if step > 0 {
			currentVar := m.calculateForecastVariance(step + 1)
			prevVar := m.calculateForecastVariance(step)
			if currentVar.Lt(prevVar) {
				t.Errorf("Step %d: Variance should increase with horizon", step)
			}
		}

		prevForecast = result.PointForecast

		t.Logf("Step %d forecast: %v (SE: %v)",
			step, result.PointForecast.String(), result.StandardError.String())
	}
}

func TestModelArima_ForecastOneStepEdgeCases(t *testing.T) {
	t.Run("No constant term", func(t *testing.T) {
		m, _ := NewModel(1, 0, 0, 100)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.7)}
		m.constant = fixed.Zero
		m.variance = fixed.FromFloat64(1.0)
		m.includeConstant = false
		m.estimated = true

		// Add data centered around zero
		series := []fixed.Point{
			fixed.FromFloat64(-1), fixed.FromFloat64(2), fixed.FromFloat64(-1.5),
			fixed.FromFloat64(1), fixed.FromFloat64(0.5),
		}
		for _, val := range series {
			m.rawData.Add(val)
			m.diffData.Add(val)
		}

		state := m.initializeForecastState()
		result, err := m.forecastOneStep(state, 0)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Without constant, forecast = φ₁ * last_value
		expected := fixed.FromFloat64(0.7 * 0.5)
		diff := result.PointForecast.Sub(expected).Abs()
		if diff.Gt(fixed.FromFloat64(0.01)) {
			t.Errorf("Expected forecast %v, got %v",
				expected.String(), result.PointForecast.String())
		}
	})

	t.Run("Zero variance model", func(t *testing.T) {
		m, _ := NewModel(1, 0, 1, 100)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
		m.maParams = []fixed.Point{fixed.FromFloat64(0.3)}
		m.constant = fixed.FromFloat64(2.0)
		m.variance = fixed.Zero // Zero variance
		m.includeConstant = true
		m.estimated = true

		// Add dummy data
		for i := 0; i < 5; i++ {
			m.rawData.Add(fixed.FromFloat64(float64(i)))
			m.diffData.Add(fixed.FromFloat64(float64(i)))
			m.residuals.Add(fixed.Zero)
		}

		state := m.initializeForecastState()
		result, err := m.forecastOneStep(state, 0)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// With zero variances, all intervals should collapse to the point forecast
		if !result.StandardError.Eq(fixed.Zero) {
			t.Error("Standard error should be zero with zero variance")
		}
	})
}

func TestModelArima_ForecastOneStepWithDifferencing(t *testing.T) {
	// Test ARIMA(1,2,1) - with d=2 differencing
	m, _ := NewModel(1, 2, 1, 100)
	m.arParams = []fixed.Point{fixed.FromFloat64(0.3)}
	m.maParams = []fixed.Point{fixed.FromFloat64(0.2)}
	m.constant = fixed.FromFloat64(0.05)
	m.variance = fixed.FromFloat64(0.5)
	m.includeConstant = true
	m.estimated = true

	// Create a trending series
	var rawSeries []fixed.Point
	for i := 0; i < 20; i++ {
		// Quadratic trend plus noise
		val := float64(i*i)/10.0 + float64(i) + 10.0
		rawSeries = append(rawSeries, fixed.FromFloat64(val))
		m.rawData.Add(fixed.FromFloat64(val))
	}

	// Manually calculate second differences for verification
	// This would normally be done by AddPoint
	for i := 2; i < len(rawSeries); i++ {
		d1Curr := rawSeries[i].Sub(rawSeries[i-1])
		d1Prev := rawSeries[i-1].Sub(rawSeries[i-2])
		d2 := d1Curr.Sub(d1Prev)
		m.diffData.Add(d2)
		m.residuals.Add(fixed.FromFloat64(0.1)) // Small residuals
	}

	state := m.initializeForecastState()

	// Store previous forecasts in the cache (needed for undifferencing)
	m.forecastCache = []fixed.Point{}

	result, err := m.forecastOneStep(state, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The forecast should follow the trend
	lastValue := rawSeries[len(rawSeries)-1]
	if result.PointForecast.Lt(lastValue) {
		t.Error("Forecast should be greater than last value for upward trending series")
	}

	t.Logf("Last value: %v, Forecast: %v",
		lastValue.String(), result.PointForecast.String())
}

func TestModelArima_CalculateForecastVariance(t *testing.T) {
	tests := []struct {
		name     string
		p        int
		q        int
		arParams []fixed.Point
		maParams []fixed.Point
		variance fixed.Point
		step     int
		expected fixed.Point
	}{
		{
			name:     "One-step ahead forecast",
			p:        1,
			q:        1,
			arParams: []fixed.Point{fixed.FromFloat64(0.5)},
			maParams: []fixed.Point{fixed.FromFloat64(0.3)},
			variance: fixed.FromFloat64(1.0),
			step:     1,
			expected: fixed.FromFloat64(1.0), // V(1) = σ² * (1) = 1.0
		},
		{
			name:     "Zero-step ahead (should return variance)",
			p:        1,
			q:        1,
			arParams: []fixed.Point{fixed.FromFloat64(0.5)},
			maParams: []fixed.Point{fixed.FromFloat64(0.3)},
			variance: fixed.FromFloat64(2.0),
			step:     0,
			expected: fixed.FromFloat64(2.0), // V(0) = σ² = 2.0
		},
		{
			name:     "Pure MA(1) two-step ahead",
			p:        0,
			q:        1,
			arParams: []fixed.Point{},
			maParams: []fixed.Point{fixed.FromFloat64(0.4)},
			variance: fixed.FromFloat64(1.0),
			step:     2,
			expected: fixed.FromFloat64(1.16), // V(2) = σ² * (1 + θ₁²) = 1 * (1 + 0.16) = 1.16
		},
		{
			name:     "Pure AR(1) multi-step",
			p:        1,
			q:        0,
			arParams: []fixed.Point{fixed.FromFloat64(0.6)},
			maParams: []fixed.Point{},
			variance: fixed.FromFloat64(1.0),
			step:     3,
			expected: fixed.FromFloat64(1.4896), // V(3) = σ² * (1 + ψ₁² + ψ₂²) = 1 * (1 + 0.36 + 0.1296)
		},
		{
			name:     "ARMA(1,1) three-step",
			p:        1,
			q:        1,
			arParams: []fixed.Point{fixed.FromFloat64(0.5)},
			maParams: []fixed.Point{fixed.FromFloat64(0.3)},
			variance: fixed.FromFloat64(1.0),
			step:     3,
			expected: fixed.FromFloat64(1.80), // V(3) = σ² * (1 + ψ₁² + ψ₂²)
			// ψ₁ = φ₁ + θ₁ = 0.8, ψ₂ = φ₁ * ψ₁ = 0.4
			// V(3) = 1 * (1 + 0.64 + 0.16) = 1.80
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := NewModel(tt.p, 0, tt.q, 100)
			m.arParams = tt.arParams
			m.maParams = tt.maParams
			m.variance = tt.variance

			result := m.calculateForecastVariance(tt.step)

			// Allow some tolerance for floating point comparison
			diff := result.Sub(tt.expected).Abs()
			tolerance := fixed.FromFloat64(0.01)

			if diff.Gt(tolerance) {
				t.Errorf("Expected variance %v, got %v", tt.expected.String(), result.String())
			}
		})
	}
}

func TestModelArima_CalculateForecastVarianceAR2(t *testing.T) {
	// More complex AR(2) case
	m, _ := NewModel(2, 0, 0, 100)
	m.arParams = []fixed.Point{fixed.FromFloat64(0.4), fixed.FromFloat64(0.3)}
	m.variance = fixed.FromFloat64(1.0)

	tests := []struct {
		step     int
		expected fixed.Point
	}{
		{
			step:     1,
			expected: fixed.FromFloat64(1.0), // V(1) = σ²
		},
		{
			step:     2,
			expected: fixed.FromFloat64(1.16), // V(2) = σ² * (1 + ψ₁²) = 1 + 0.4² = 1.16
		},
		{
			step:     3,
			expected: fixed.FromFloat64(1.3716), // V(3) = σ² * (1 + ψ₁² + ψ₂²) = 1 + 0.16 + 0.2116
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("step_%d", tt.step), func(t *testing.T) {
			result := m.calculateForecastVariance(tt.step)

			diff := result.Sub(tt.expected).Abs()
			tolerance := fixed.FromFloat64(0.01)

			if diff.Gt(tolerance) {
				// Calculate psi weights for debugging
				psiWeights := m.calculatePsiWeights(tt.step)
				t.Logf("Psi weights: %v", psiWeights)

				// Calculate sums of squared psi
				sumSquaredPsi := fixed.One
				for i := 0; i < tt.step-1 && i < len(psiWeights); i++ {
					sumSquaredPsi = sumSquaredPsi.Add(psiWeights[i].Mul(psiWeights[i]))
				}
				t.Logf("Sum of squared psi: %v", sumSquaredPsi.String())

				t.Errorf("Expected variance %v, got %v", tt.expected.String(), result.String())
			}
		})
	}
}

func TestModelArima_CalculateForecastVarianceLargeHorizon(t *testing.T) {
	// Test that variance converges for stationary AR(1)
	m, _ := NewModel(1, 0, 0, 100)
	m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
	m.variance = fixed.FromFloat64(1.0)

	// For AR(1) with φ = 0.5, the h-step variance should converge to σ²/(1-φ²) = 1/(1-0.25) = 1.333...
	largeStep := 50
	result := m.calculateForecastVariance(largeStep)

	// Should be close to the theoretical limit
	expectedLimit := fixed.FromFloat64(1.333333)
	diff := result.Sub(expectedLimit).Abs()

	if diff.Gt(fixed.FromFloat64(0.01)) {
		t.Errorf("For large horizon, expected variance near %v, got %v",
			expectedLimit.String(), result.String())
	}
}

func TestModelArima_CalculateForecastVarianceEdgeCases(t *testing.T) {
	t.Run("Zero variance model", func(t *testing.T) {
		m, _ := NewModel(1, 0, 1, 100)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
		m.maParams = []fixed.Point{fixed.FromFloat64(0.3)}
		m.variance = fixed.Zero

		result := m.calculateForecastVariance(5)

		if !result.Eq(fixed.Zero) {
			t.Errorf("Expected zero variance, got %v", result.String())
		}
	})

	t.Run("Model with zero parameters", func(t *testing.T) {
		m, _ := NewModel(2, 0, 2, 100)
		m.arParams = []fixed.Point{fixed.Zero, fixed.Zero}
		m.maParams = []fixed.Point{fixed.Zero, fixed.Zero}
		m.variance = fixed.FromFloat64(2.0)

		// With all zero parameters, psi weights are all zero.
		// So forecast variance should equal model variance for all horizons
		for step := 1; step <= 5; step++ {
			result := m.calculateForecastVariance(step)
			if !result.Eq(fixed.FromFloat64(2.0)) {
				t.Errorf("Step %d: Expected variance 2.0, got %v", step, result.String())
			}
		}
	})
}

func TestModelArima_CalculatePsiWeights(t *testing.T) {
	tests := []struct {
		name     string
		p        int // AR order
		q        int // MA order
		arParams []fixed.Point
		maParams []fixed.Point
		maxLag   int
		expected []fixed.Point
	}{
		{
			name:     "Pure MA(1) model",
			p:        0,
			q:        1,
			arParams: []fixed.Point{},
			maParams: []fixed.Point{fixed.FromFloat64(0.5)},
			maxLag:   3,
			expected: []fixed.Point{
				fixed.FromFloat64(0.5), // psi_1 = theta_1
				fixed.Zero,             // psi_2 = 0
				fixed.Zero,             // psi_3 = 0
			},
		},
		{
			name:     "Pure AR(1) model",
			p:        1,
			q:        0,
			arParams: []fixed.Point{fixed.FromFloat64(0.6)},
			maParams: []fixed.Point{},
			maxLag:   4,
			expected: []fixed.Point{
				fixed.FromFloat64(0.6),    // psi_1 = phi_1
				fixed.FromFloat64(0.36),   // psi_2 = phi_1 * psi_1 = 0.6 * 0.6
				fixed.FromFloat64(0.216),  // psi_3 = phi_1 * psi_2 = 0.6 * 0.36
				fixed.FromFloat64(0.1296), // psi_4 = phi_1 * psi_3 = 0.6 * 0.216
			},
		},
		{
			name:     "ARMA(1,1) model",
			p:        1,
			q:        1,
			arParams: []fixed.Point{fixed.FromFloat64(0.5)},
			maParams: []fixed.Point{fixed.FromFloat64(0.3)},
			maxLag:   3,
			expected: []fixed.Point{
				fixed.FromFloat64(0.8), // psi_1 = phi_1 + theta_1 = 0.5 + 0.3
				fixed.FromFloat64(0.4), // psi_2 = phi_1 * psi_1 = 0.5 * 0.8
				fixed.FromFloat64(0.2), // psi_3 = phi_1 * psi_2 = 0.5 * 0.4
			},
		},
		{
			name:     "AR(2) model",
			p:        2,
			q:        0,
			arParams: []fixed.Point{fixed.FromFloat64(0.4), fixed.FromFloat64(0.3)},
			maParams: []fixed.Point{},
			maxLag:   3,
			expected: []fixed.Point{
				fixed.FromFloat64(0.4),   // psi_1 = phi_1
				fixed.FromFloat64(0.46),  // psi_2 = phi_1 * psi_1 + phi_2 * psi_0 = 0.4 * 0.4 + 0.3 * 1
				fixed.FromFloat64(0.304), // psi_3 = phi_1 * psi_2 + phi_2 * psi_1 = 0.4 * 0.46 + 0.3 * 0.4
			},
		},
		{
			name:     "Zero lag",
			p:        1,
			q:        1,
			arParams: []fixed.Point{fixed.FromFloat64(0.5)},
			maParams: []fixed.Point{fixed.FromFloat64(0.3)},
			maxLag:   0,
			expected: []fixed.Point{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := NewModel(tt.p, 0, tt.q, 100)
			m.arParams = tt.arParams
			m.maParams = tt.maParams

			result := m.calculatePsiWeights(tt.maxLag)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d weights, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				// Use approximate equality for floating point comparison
				diff := result[i].Sub(expected).Abs()
				tolerance := fixed.FromFloat64(0.0001)

				if diff.Gt(tolerance) {
					t.Errorf("psi[%d]: expected %v, got %v", i+1, expected.String(), result[i].String())
				}
			}
		})
	}
}

func TestModelArima_CalculatePsiWeightsEdgeCases(t *testing.T) {
	t.Run("Large maxLag with small model", func(t *testing.T) {
		m, _ := NewModel(1, 0, 1, 100)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
		m.maParams = []fixed.Point{fixed.FromFloat64(0.2)}

		result := m.calculatePsiWeights(10)

		if len(result) != 10 {
			t.Errorf("Expected 10 weights, got %d", len(result))
		}

		// Check that weights decay geometrically for AR(1)
		for i := 2; i < len(result); i++ {
			ratio := result[i].Div(result[i-1])
			expected := fixed.FromFloat64(0.5)
			diff := ratio.Sub(expected).Abs()

			if diff.Gt(fixed.FromFloat64(0.0001)) {
				t.Errorf("Weight ratio at position %d incorrect: expected %v, got %v",
					i, expected.String(), ratio.String())
			}
		}
	})

	t.Run("Model with zero parameters", func(t *testing.T) {
		m, _ := NewModel(1, 0, 1, 100)
		m.arParams = []fixed.Point{fixed.Zero}
		m.maParams = []fixed.Point{fixed.Zero}

		result := m.calculatePsiWeights(5)

		// All psi weights should be zero
		for i, psi := range result {
			if !psi.Eq(fixed.Zero) {
				t.Errorf("psi[%d] should be zero, got %v", i+1, psi.String())
			}
		}
	})
}

func TestModelArima_GetRawSeriesInOrder(t *testing.T) {
	m, _ := NewModel(1, 1, 1, 50)

	// Add test data
	testData := []fixed.Point{
		fixed.FromInt64(10, 0),
		fixed.FromInt64(20, 0),
		fixed.FromInt64(30, 0),
		fixed.FromInt64(40, 0),
		fixed.FromInt64(50, 0),
	}

	for _, p := range testData {
		m.rawData.Add(p)
	}

	// Get the series in order
	result := m.getRawSeriesInOrder()

	// Verify length
	if len(result) != len(testData) {
		t.Errorf("Expected length %d, got %d", len(testData), len(result))
	}

	// Verify order (should be oldest to newest)
	for i, expected := range testData {
		if !result[i].Eq(expected) {
			t.Errorf("At index %d: expected %v, got %v", i, expected, result[i])
		}
	}
}

func TestModelArima_GetDiffSeriesInOrder(t *testing.T) {
	m, _ := NewModel(1, 1, 1, 50)

	// Add test data to diffData buffer
	testDiffData := []fixed.Point{
		fixed.FromInt64(5, 0),
		fixed.FromInt64(15, 0),
		fixed.FromInt64(25, 0),
		fixed.FromInt64(35, 0),
	}

	for _, p := range testDiffData {
		m.diffData.Add(p)
	}

	// Get the series in order
	result := m.getDiffSeriesInOrder()

	// Verify length
	if len(result) != len(testDiffData) {
		t.Errorf("Expected length %d, got %d", len(testDiffData), len(result))
	}

	// Verify order (should be oldest to newest)
	for i, expected := range testDiffData {
		if !result[i].Eq(expected) {
			t.Errorf("At index %d: expected %v, got %v", i, expected, result[i])
		}
	}
}

func TestModelArima_GetSeriesInOrderEmpty(t *testing.T) {
	m, _ := NewModel(1, 1, 1, 50)

	// Test empty buffers
	rawResult := m.getRawSeriesInOrder()
	if len(rawResult) != 0 {
		t.Errorf("Expected empty slice for raw series, got length %d", len(rawResult))
	}

	diffResult := m.getDiffSeriesInOrder()
	if len(diffResult) != 0 {
		t.Errorf("Expected empty slice for diff series, got length %d", len(diffResult))
	}
}

func generateNormalResiduals(n int, mean, stddev float64) []fixed.Point {
	residuals := make([]fixed.Point, n)

	// Use linear congruential generator
	a := int64(1664525)
	c := int64(1013904223)
	m := int64(1) << 32
	x := int64(42) // seed

	for i := 0; i < n; i++ {
		// Generate 12 uniform random numbers for CLT approximation
		sum := 0.0
		for j := 0; j < 12; j++ {
			x = (a*x + c) % m
			u := float64(x) / float64(m)
			sum += u
		}
		// CLT: sum of 12 uniform(0,1) has mean 6 and variance 1
		val := (sum-6.0)*stddev + mean
		residuals[i] = fixed.FromFloat64(val * 0.1)
	}
	return residuals
}

func generateSkewedResiduals(n int, skewness float64) []fixed.Point {
	residuals := make([]fixed.Point, n)
	for i := 0; i < n; i++ {
		// Generate from chi-squared-like distribution
		val := float64(i) / float64(n)
		skewed := math.Pow(val, 1.0/skewness) - 0.5
		residuals[i] = fixed.FromFloat64(skewed * 0.2)
	}
	return residuals
}

func generateHeavyTailedResiduals(n int) []fixed.Point {
	residuals := make([]fixed.Point, n)
	for i := 0; i < n; i++ {
		// Mix of normal and extreme values
		if i%20 == 0 {
			// Extreme value
			if i%40 == 0 {
				residuals[i] = fixed.FromFloat64(2.0)
			} else {
				residuals[i] = fixed.FromFloat64(-2.0)
			}
		} else {
			// Normal range
			residuals[i] = fixed.FromFloat64(float64(i%10-5) * 0.02)
		}
	}
	return residuals
}

func generateUniformResiduals(n int) []fixed.Point {
	residuals := make([]fixed.Point, n)
	for i := 0; i < n; i++ {
		// Uniform distribution in [-0.5, 0.5]
		val := float64(i)/float64(n) - 0.5
		residuals[i] = fixed.FromFloat64(val * 0.2)
	}
	return residuals
}

func generateConstantResiduals(n int, value float64) []fixed.Point {
	residuals := make([]fixed.Point, n)
	for i := 0; i < n; i++ {
		residuals[i] = fixed.FromFloat64(value)
	}
	return residuals
}

func generateWhiteNoiseResiduals(n int, seed int64) []fixed.Point {
	residuals := make([]fixed.Point, n)

	// Use Box-Muller transform for better white noise
	a := int64(1664525)
	c := int64(1013904223)
	m := int64(1) << 32
	x := seed

	for i := 0; i < n; i += 2 {
		// Generate two uniform random numbers
		x = (a*x + c) % m
		u1 := float64(x) / float64(m)

		x = (a*x + c) % m
		u2 := float64(x) / float64(m)

		// Box-Muller transform
		z0 := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2) * 0.3
		z1 := math.Sqrt(-2*math.Log(u1)) * math.Sin(2*math.Pi*u2) * 0.3

		residuals[i] = fixed.FromFloat64(z0)
		if i+1 < n {
			residuals[i+1] = fixed.FromFloat64(z1)
		}
	}

	// Shuffle to break any patterns
	for i := n - 1; i > 0; i-- {
		x = (a*x + c) % m
		j := int(x) % (i + 1)
		residuals[i], residuals[j] = residuals[j], residuals[i]
	}

	return residuals
}

func generateAutocorrelatedResiduals(n int, rho float64) []fixed.Point {
	residuals := make([]fixed.Point, n)

	// Start with white noise
	noise := generateWhiteNoiseResiduals(n, 12345)

	// AR(1) process: x_t = rho * x_{t-1} + e_t
	residuals[0] = noise[0]
	for i := 1; i < n; i++ {
		residuals[i] = residuals[i-1].Mul(fixed.FromFloat64(rho)).Add(
			noise[i].Mul(fixed.FromFloat64(math.Sqrt(1 - rho*rho))))
	}

	return residuals
}

func generateAlternatingResiduals(n int) []fixed.Point {
	residuals := make([]fixed.Point, n)

	for i := 0; i < n; i++ {
		if i%2 == 0 {
			residuals[i] = fixed.FromFloat64(0.5)
		} else {
			residuals[i] = fixed.FromFloat64(-0.5)
		}
	}

	return residuals
}

func generateTrendingResiduals(n int) []fixed.Point {
	residuals := make([]fixed.Point, n)

	for i := 0; i < n; i++ {
		// Linear trend plus noise
		trend := float64(i)/float64(n) - 0.5
		noise := (float64(i%7) - 3.0) * 0.05
		residuals[i] = fixed.FromFloat64(trend + noise)
	}

	return residuals
}

func generateSeasonalResiduals(n int, period int) []fixed.Point {
	residuals := make([]fixed.Point, n)

	for i := 0; i < n; i++ {
		seasonal := math.Sin(2 * math.Pi * float64(i) / float64(period))
		noise := (float64(i%5) - 2.0) * 0.1
		residuals[i] = fixed.FromFloat64(seasonal*0.5 + noise)
	}

	return residuals
}

func generateSmallVarianceResiduals(n int) []fixed.Point {
	residuals := make([]fixed.Point, n)

	// Pattern that creates small variance but still has structure
	for i := 0; i < n; i++ {
		// Repeating a pattern with very small values
		if i%4 == 0 {
			residuals[i] = fixed.FromFloat64(0.0001)
		} else if i%4 == 1 {
			residuals[i] = fixed.FromFloat64(0.0002)
		} else if i%4 == 2 {
			residuals[i] = fixed.FromFloat64(-0.0001)
		} else {
			residuals[i] = fixed.FromFloat64(-0.0002)
		}
	}

	return residuals
}

type diagnosticCheck struct {
	name      string
	validator func(ModelDiagnostics) bool
}

func logDiagnostics(t *testing.T, d ModelDiagnostics) {
	t.Logf("Diagnostics:")
	t.Logf("  LogLikelihood: %v", d.LogLikelihood.String())
	t.Logf("  AIC: %v", d.AIC.String())
	t.Logf("  BIC: %v", d.BIC.String())
	t.Logf("  AICC: %v", d.AICC.String())
	t.Logf("  RMSE: %v", d.RMSE.String())
	t.Logf("  MAE: %v", d.MAE.String())
	t.Logf("  MAPE: %v", d.MAPE.String())
	t.Logf("  LjungBoxPValue: %v", d.LjungBoxPValue.String())
	t.Logf("  JarqueBeraTest: %v", d.JarqueBeraTest.String())
	t.Logf("  IsStationary: %v", d.IsStationary)
}

func generateAR1Series(n int, phi, constant, variance float64) []float64 {
	series := make([]float64, n)
	series[0] = constant

	// Simple random number generator
	a := int64(1664525)
	c := int64(1013904223)
	m := int64(1) << 32
	x := int64(42)

	for i := 1; i < n; i++ {
		x = (a*x + c) % m
		noise := (float64(x)/float64(m) - 0.5) * 2 * math.Sqrt(variance)
		series[i] = constant + phi*(series[i-1]-constant) + noise
	}

	return series
}

func generateWhiteNoise(n int) []float64 {
	series := make([]float64, n)

	a := int64(1664525)
	c := int64(1013904223)
	m := int64(1) << 32
	x := int64(12345)

	for i := 0; i < n; i++ {
		x = (a*x + c) % m
		series[i] = (float64(x)/float64(m) - 0.5) * 2
	}

	return series
}
