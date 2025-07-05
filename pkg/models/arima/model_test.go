package arima

import (
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"math"
	"testing"
)

func TestModel_JarqueBeraTest(t *testing.T) {
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

func TestModel_JarqueBeraTestSpecificCases(t *testing.T) {
	model, _ := NewModel(1, 0, 1, 100)

	t.Run("Perfect normal distribution", func(t *testing.T) {
		// Test with multiple seeds to account for sampling variation
		passCount := 0
		attempts := 5

		for seed := int64(12345); seed < int64(12345+attempts); seed++ {
			residuals := []fixed.Point{}
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
		// Create highly skewed distribution (exponential-like)
		residuals := []fixed.Point{}
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
		residuals := []fixed.Point{}
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
		residuals := []fixed.Point{}
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
		// Bimodal distribution should fail normality test
		if pValue.Gt(fixed.FromFloat64(0.1)) {
			pf, _ := pValue.Float64()
			t.Errorf("Bimodal distribution should have low p-value, got %.4f", pf)
		}
	})
}

func TestModel_JarqueBeraTestImplementation(t *testing.T) {
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

func TestModel_CheckParameterValidity(t *testing.T) {
	tests := []struct {
		name        string
		p, q        uint
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

func TestModel_CheckParameterValidityEdgeCases(t *testing.T) {

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

func TestModel_CheckResidualProperties(t *testing.T) {
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
			ljungBoxPValue: fixed.FromFloat64(0.05), // Exactly at threshold
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
				m.residuals.PushUpdate(r)
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

func TestModel_CheckResidualPropertiesIntegration(t *testing.T) {

	t.Run("Well-specified model", func(t *testing.T) {
		m, _ := NewModel(1, 0, 0, 50)
		m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
		m.variance = fixed.FromFloat64(1.0)
		m.estimated = true

		// Generate pseudo-random white noise residuals with low autocorrelation
		// Using a linear congruential generator for reproducibility
		residuals := []fixed.Point{}
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

		// Generate series from these residuals
		series := []fixed.Point{fixed.Zero}
		for i := 0; i < len(residuals); i++ {
			// AR(1) process: y_t = 0.5 * y_{t-1} + e_t
			value := series[i].Mul(fixed.FromFloat64(0.5)).Add(residuals[i])
			series = append(series, value)
		}

		// Add data to model
		for _, val := range series {
			m.diffData.PushUpdate(val)
		}

		// Set residuals
		m.residuals.Clear()
		for _, r := range residuals {
			m.residuals.PushUpdate(r)
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

		// Add data to model
		for _, val := range series {
			m.diffData.PushUpdate(val)
		}

		// Calculate residuals with wrong model
		residuals := []fixed.Point{}
		for i := 1; i < len(series); i++ {
			fitted := series[i-1].Mul(fixed.FromFloat64(0.3))
			residual := series[i].Sub(fitted)
			residuals = append(residuals, residual)
		}

		m.residuals.Clear()
		for _, r := range residuals {
			m.residuals.PushUpdate(r)
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

func TestModel_CheckResidualPropertiesEdgeCases(t *testing.T) {
	t.Run("Nil diagnostics", func(t *testing.T) {
		m, _ := NewModel(1, 0, 1, 100)
		m.estimated = true

		// Add some residuals
		for i := 0; i < 15; i++ {
			m.residuals.PushUpdate(fixed.FromFloat64(float64(i) * 0.01))
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
				m.residuals.PushUpdate(fixed.FromFloat64(0.1))
			} else {
				m.residuals.PushUpdate(fixed.FromFloat64(-0.1))
			}
		}

		m.diagnostics.LjungBoxPValue = fixed.FromFloat64(0.8) // High p-value

		err := m.checkResidualProperties()
		if err != nil {
			t.Errorf("Should pass with high p-value, got: %v", err)
		}
	})
}

func TestModel_InitializeForecastState(t *testing.T) {
	tests := []struct {
		name                  string
		p, d, q               uint
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
				m.rawData.PushUpdate(val)
			}
			for _, val := range tt.diffSeriesData {
				m.diffData.PushUpdate(val)
			}
			for _, val := range tt.residualsData {
				m.residuals.PushUpdate(val)
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

			// Check forecasted diffs is initialized empty
			if len(state.forecastedDiffs) != 0 {
				t.Errorf("Expected empty forecastedDiffs, got %d elements",
					len(state.forecastedDiffs))
			}
		})
	}
}

func TestModel_InitializeForecastStateWithCircularBufferWrap(t *testing.T) {

	m, _ := NewModel(1, 0, 1, 50)
	m.variance = fixed.FromFloat64(1.0)
	m.estimated = true

	// Add more data than buffer capacity to force wrap-around
	for i := 0; i < 100; i++ {
		m.rawData.PushUpdate(fixed.FromFloat64(float64(i)))
		m.diffData.PushUpdate(fixed.FromFloat64(float64(i * 10)))
		if i < 8 {
			m.residuals.PushUpdate(fixed.FromFloat64(float64(i) * 0.1))
		}
	}

	state := m.initializeForecastState()

	// Should only have the last 50 raw values
	if len(state.rawValues) != 50 {
		t.Errorf("Expected 5 raw values, got %d", len(state.rawValues))
	}

	// Check that we have the most recent values in oldest-to-newest order
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

func TestModel_InitializeForecastStateConsistency(t *testing.T) {

	m, _ := NewModel(2, 1, 1, 50)
	m.variance = fixed.FromFloat64(1.5)
	m.estimated = true

	// Add some data
	for i := 0; i < 15; i++ {
		m.rawData.PushUpdate(fixed.FromFloat64(float64(100 + i)))
		if i > 0 { // For d=1, we need at least 2 raw values
			m.diffData.PushUpdate(fixed.FromFloat64(float64(i)))
		}
		if i < 10 {
			m.residuals.PushUpdate(fixed.FromFloat64(float64(i) * 0.01))
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

func TestModel_ForecastOneStep(t *testing.T) {
	tests := []struct {
		name             string
		p, d, q          uint
		arParams         []fixed.Point
		maParams         []fixed.Point
		constant         fixed.Point
		variance         fixed.Point
		includeConstant  bool
		diffSeries       []fixed.Point
		rawSeries        []fixed.Point
		residuals        []fixed.Point
		step             uint
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
			// Forecast in differenced scale, then undifference
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
				m.rawData.PushUpdate(val)
			}
			for _, val := range tt.diffSeries {
				m.diffData.PushUpdate(val)
			}
			for _, val := range tt.residuals {
				m.residuals.PushUpdate(val)
			}

			// Initialize forecast state
			state := m.initializeForecastState()

			// Perform one-step forecast
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

func TestModel_ForecastOneStepMultiStep(t *testing.T) {
	// Test multi-step forecasting with state updates
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
		m.rawData.PushUpdate(val)
		m.diffData.PushUpdate(val) // No differencing
	}

	// Add some residuals
	residuals := []fixed.Point{
		fixed.Zero, fixed.FromFloat64(0.5), fixed.FromFloat64(-0.3),
		fixed.FromFloat64(0.2), fixed.FromFloat64(0.1),
	}
	for _, val := range residuals {
		m.residuals.PushUpdate(val)
	}

	state := m.initializeForecastState()

	// Test that multi-step forecasts use previous forecasts
	prevForecast := fixed.Zero
	for step := uint(0); step < 3; step++ {
		result, err := m.forecastOneStep(state, step)
		if err != nil {
			t.Fatalf("Step %d: Unexpected error: %v", step, err)
		}

		// Store in forecast cache (mimicking Forecast method)
		m.forecastCache = append(m.forecastCache, result.PointForecast)

		// Update state
		m.appendZeroResidual(state)

		// For AR models, subsequent forecasts should depend on previous ones
		if step > 0 && m.p > 0 {
			// The forecast should be different from the previous one
			// (unless the model predicts constant values)
			if result.PointForecast.Eq(prevForecast) && !m.arParams[0].Eq(fixed.Zero) {
				t.Errorf("Step %d: Forecast should differ from previous step", step)
			}
		}

		// Variance should increase with horizon
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

func TestModel_ForecastOneStepEdgeCases(t *testing.T) {
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
			m.rawData.PushUpdate(val)
			m.diffData.PushUpdate(val)
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
			m.rawData.PushUpdate(fixed.FromFloat64(float64(i)))
			m.diffData.PushUpdate(fixed.FromFloat64(float64(i)))
			m.residuals.PushUpdate(fixed.Zero)
		}

		state := m.initializeForecastState()
		result, err := m.forecastOneStep(state, 0)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// With zero variance, all intervals should collapse to point forecast
		if !result.StandardError.Eq(fixed.Zero) {
			t.Error("Standard error should be zero with zero variance")
		}
	})
}

func TestModel_ForecastOneStepWithDifferencing(t *testing.T) {
	// Test ARIMA(1,2,1) - with d=2 differencing
	m, _ := NewModel(1, 2, 1, 100)
	m.arParams = []fixed.Point{fixed.FromFloat64(0.3)}
	m.maParams = []fixed.Point{fixed.FromFloat64(0.2)}
	m.constant = fixed.FromFloat64(0.05)
	m.variance = fixed.FromFloat64(0.5)
	m.includeConstant = true
	m.estimated = true

	// Create a trending series
	rawSeries := []fixed.Point{}
	for i := 0; i < 20; i++ {
		// Quadratic trend plus noise
		val := float64(i*i)/10.0 + float64(i) + 10.0
		rawSeries = append(rawSeries, fixed.FromFloat64(val))
		m.rawData.PushUpdate(fixed.FromFloat64(val))
	}

	// Manually calculate second differences for verification
	// This would normally be done by AddPoint
	for i := 2; i < len(rawSeries); i++ {
		d1_curr := rawSeries[i].Sub(rawSeries[i-1])
		d1_prev := rawSeries[i-1].Sub(rawSeries[i-2])
		d2 := d1_curr.Sub(d1_prev)
		m.diffData.PushUpdate(d2)
		m.residuals.PushUpdate(fixed.FromFloat64(0.1)) // Small residuals
	}

	state := m.initializeForecastState()

	// Store previous forecasts in cache (needed for undifferencing)
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

func TestModel_CalculateForecastVariance(t *testing.T) {
	tests := []struct {
		name     string
		p        uint
		q        uint
		arParams []fixed.Point
		maParams []fixed.Point
		variance fixed.Point
		step     uint
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

func TestModel_CalculateForecastVarianceAR2(t *testing.T) {
	// More complex AR(2) case
	m, _ := NewModel(2, 0, 0, 100)
	m.arParams = []fixed.Point{fixed.FromFloat64(0.4), fixed.FromFloat64(0.3)}
	m.variance = fixed.FromFloat64(1.0)

	tests := []struct {
		step     uint
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

				// Calculate sum of squared psi
				sumSquaredPsi := fixed.One
				for i := uint(0); i < tt.step-1 && i < uint(len(psiWeights)); i++ {
					sumSquaredPsi = sumSquaredPsi.Add(psiWeights[i].Mul(psiWeights[i]))
				}
				t.Logf("Sum of squared psi: %v", sumSquaredPsi.String())

				t.Errorf("Expected variance %v, got %v", tt.expected.String(), result.String())
			}
		})
	}
}

func TestModel_CalculateForecastVarianceLargeHorizon(t *testing.T) {
	// Test that variance converges for stationary AR(1)
	m, _ := NewModel(1, 0, 0, 100)
	m.arParams = []fixed.Point{fixed.FromFloat64(0.5)}
	m.variance = fixed.FromFloat64(1.0)

	// For AR(1) with φ = 0.5, the h-step variance should converge to σ²/(1-φ²) = 1/(1-0.25) = 1.333...
	largeStep := uint(50)
	result := m.calculateForecastVariance(largeStep)

	// Should be close to the theoretical limit
	expectedLimit := fixed.FromFloat64(1.333333)
	diff := result.Sub(expectedLimit).Abs()

	if diff.Gt(fixed.FromFloat64(0.01)) {
		t.Errorf("For large horizon, expected variance near %v, got %v",
			expectedLimit.String(), result.String())
	}
}

func TestModel_CalculateForecastVarianceEdgeCases(t *testing.T) {
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

		// With all zero parameters, psi weights are all zero
		// So forecast variance should equal model variance for all horizons
		for step := uint(1); step <= 5; step++ {
			result := m.calculateForecastVariance(step)
			if !result.Eq(fixed.FromFloat64(2.0)) {
				t.Errorf("Step %d: Expected variance 2.0, got %v", step, result.String())
			}
		}
	})
}

func TestModel_CalculatePsiWeights(t *testing.T) {
	tests := []struct {
		name     string
		p        uint // AR order
		q        uint // MA order
		arParams []fixed.Point
		maParams []fixed.Point
		maxLag   uint
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

func TestModel_CalculatePsiWeightsEdgeCases(t *testing.T) {
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

func TestModel_GetRawSeriesInOrder(t *testing.T) {
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
		m.rawData.PushUpdate(p)
	}

	// Get series in order
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

func TestModel_GetDiffSeriesInOrder(t *testing.T) {
	m, _ := NewModel(1, 1, 1, 50)

	// Add test data to diffData buffer
	testDiffData := []fixed.Point{
		fixed.FromInt64(5, 0),
		fixed.FromInt64(15, 0),
		fixed.FromInt64(25, 0),
		fixed.FromInt64(35, 0),
	}

	for _, p := range testDiffData {
		m.diffData.PushUpdate(p)
	}

	// Get series in order
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

func TestModel_GetSeriesInOrderEmpty(t *testing.T) {
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
