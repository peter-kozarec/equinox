package fixed

import (
	"testing"
)

func createPoints(values ...float64) []Point {
	points := make([]Point, len(values))
	for i, v := range values {
		points[i] = FromFloat64(v)
	}
	return points
}

func assertPointEqual(t *testing.T, expected, actual Point, tolerance float64, msg string) {
	t.Helper()
	diff := expected.Sub(actual).Abs()
	tol := FromFloat64(tolerance)
	if diff.Gt(tol) {
		t.Errorf("%s: expected %v, got %v (diff: %v)", msg, expected, actual, diff)
	}
}

func TestFixedMath_Mean(t *testing.T) {
	tests := []struct {
		name     string
		points   []Point
		expected Point
	}{
		{
			name:     "empty slice",
			points:   []Point{},
			expected: Zero,
		},
		{
			name:     "single point",
			points:   createPoints(5.0),
			expected: FromFloat64(5.0),
		},
		{
			name:     "multiple positive points",
			points:   createPoints(1.0, 2.0, 3.0, 4.0, 5.0),
			expected: FromFloat64(3.0),
		},
		{
			name:     "mixed positive and negative",
			points:   createPoints(-2.0, -1.0, 0.0, 1.0, 2.0),
			expected: Zero,
		},
		{
			name:     "all negative",
			points:   createPoints(-5.0, -4.0, -3.0, -2.0, -1.0),
			expected: FromFloat64(-3.0),
		},
		{
			name:     "large numbers",
			points:   createPoints(1000000.0, 2000000.0, 3000000.0),
			expected: FromFloat64(2000000.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Mean(tt.points)
			assertPointEqual(t, tt.expected, result, 0.0001, "Mean calculation")
		})
	}
}

func TestFixedMath_DownsideDev(t *testing.T) {
	tests := []struct {
		name         string
		points       []Point
		riskFreeRate Point
		expected     Point
	}{
		{
			name:         "empty slice",
			points:       []Point{},
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "single point below risk-free rate",
			points:       createPoints(-1.0),
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "all points above risk-free rate",
			points:       createPoints(1.0, 2.0, 3.0),
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "mixed points with some below risk-free rate",
			points:       createPoints(-2.0, -1.0, 1.0, 2.0),
			riskFreeRate: Zero,
			expected:     FromFloat64(1.581138830084189666),
		},
		{
			name:         "all points below risk-free rate",
			points:       createPoints(-3.0, -2.0, -1.0),
			riskFreeRate: Zero,
			expected:     FromFloat64(2.160246899469286744),
		},
		{
			name:         "non-zero risk-free rate",
			points:       createPoints(0.0, 1.0, 2.0, 3.0, 4.0),
			riskFreeRate: FromFloat64(2.5),
			expected:     FromFloat64(1.707825127659933064),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DownsideDev(tt.points, tt.riskFreeRate)
			assertPointEqual(t, tt.expected, result, 0.0001, "DownsideDev calculation")
		})
	}
}

func TestFixedMath_SampleDownsideDev(t *testing.T) {
	tests := []struct {
		name         string
		points       []Point
		riskFreeRate Point
		expected     Point
	}{
		{
			name:         "empty slice",
			points:       []Point{},
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "insufficient points",
			points:       createPoints(-1.0),
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "two points below risk-free rate",
			points:       createPoints(-2.0, -1.0),
			riskFreeRate: Zero,
			expected:     FromFloat64(2.23606797),
		},
		{
			name:         "mixed points sample calculation",
			points:       createPoints(-3.0, -1.0, 1.0, 3.0),
			riskFreeRate: Zero,
			expected:     FromFloat64(3.162277660168379332),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SampleDownsideDev(tt.points, tt.riskFreeRate)
			assertPointEqual(t, tt.expected, result, 0.0001, "SampleDownsideDev calculation")
		})
	}
}

func TestFixedMath_StdDev(t *testing.T) {
	tests := []struct {
		name     string
		points   []Point
		mean     Point
		expected Point
	}{
		{
			name:     "empty slice",
			points:   []Point{},
			mean:     Zero,
			expected: Zero,
		},
		{
			name:     "single point",
			points:   createPoints(5.0),
			mean:     FromFloat64(5.0),
			expected: Zero,
		},
		{
			name:     "uniform distribution",
			points:   createPoints(1.0, 2.0, 3.0, 4.0, 5.0),
			mean:     FromFloat64(3.0),
			expected: FromFloat64(1.41421356), // sqrt(2)
		},
		{
			name:     "all same values",
			points:   createPoints(5.0, 5.0, 5.0, 5.0),
			mean:     FromFloat64(5.0),
			expected: Zero,
		},
		{
			name:     "two points",
			points:   createPoints(1.0, 5.0),
			mean:     FromFloat64(3.0),
			expected: FromFloat64(2.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StdDev(tt.points, tt.mean)
			assertPointEqual(t, tt.expected, result, 0.0001, "StdDev calculation")
		})
	}
}

func TestFixedMath_SampleStdDev(t *testing.T) {
	tests := []struct {
		name     string
		points   []Point
		mean     Point
		expected Point
	}{
		{
			name:     "empty slice",
			points:   []Point{},
			mean:     Zero,
			expected: Zero,
		},
		{
			name:     "single point",
			points:   createPoints(5.0),
			mean:     FromFloat64(5.0),
			expected: Zero,
		},
		{
			name:     "two points",
			points:   createPoints(1.0, 5.0),
			mean:     FromFloat64(3.0),
			expected: FromFloat64(2.82842712), // sqrt(8)
		},
		{
			name:     "uniform distribution sample",
			points:   createPoints(1.0, 2.0, 3.0, 4.0, 5.0),
			mean:     FromFloat64(3.0),
			expected: FromFloat64(1.58113883), // sqrt(2.5)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SampleStdDev(tt.points, tt.mean)
			assertPointEqual(t, tt.expected, result, 0.0001, "SampleStdDev calculation")
		})
	}
}

func TestFixedMath_Variance(t *testing.T) {
	tests := []struct {
		name     string
		points   []Point
		mean     Point
		expected Point
	}{
		{
			name:     "empty slice",
			points:   []Point{},
			mean:     Zero,
			expected: Zero,
		},
		{
			name:     "single point",
			points:   createPoints(5.0),
			mean:     FromFloat64(5.0),
			expected: Zero,
		},
		{
			name:     "two points",
			points:   createPoints(1.0, 5.0),
			mean:     FromFloat64(3.0),
			expected: FromFloat64(4.0),
		},
		{
			name:     "uniform distribution",
			points:   createPoints(1.0, 2.0, 3.0, 4.0, 5.0),
			mean:     FromFloat64(3.0),
			expected: FromFloat64(2.0),
		},
		{
			name:     "all same values",
			points:   createPoints(7.0, 7.0, 7.0),
			mean:     FromFloat64(7.0),
			expected: Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Variance(tt.points, tt.mean)
			assertPointEqual(t, tt.expected, result, 0.0001, "Variance calculation")
		})
	}
}

func TestFixedMath_SampleVariance(t *testing.T) {
	tests := []struct {
		name     string
		points   []Point
		mean     Point
		expected Point
	}{
		{
			name:     "empty slice",
			points:   []Point{},
			mean:     Zero,
			expected: Zero,
		},
		{
			name:     "single point",
			points:   createPoints(5.0),
			mean:     FromFloat64(5.0),
			expected: Zero,
		},
		{
			name:     "two points",
			points:   createPoints(1.0, 5.0),
			mean:     FromFloat64(3.0),
			expected: FromFloat64(8.0),
		},
		{
			name:     "uniform distribution sample",
			points:   createPoints(1.0, 2.0, 3.0, 4.0, 5.0),
			mean:     FromFloat64(3.0),
			expected: FromFloat64(2.5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SampleVariance(tt.points, tt.mean)
			assertPointEqual(t, tt.expected, result, 0.0001, "SampleVariance calculation")
		})
	}
}

func TestFixedMath_SharpeRatio(t *testing.T) {
	tests := []struct {
		name         string
		points       []Point
		riskFreeRate Point
		expected     Point
	}{
		{
			name:         "empty slice",
			points:       []Point{},
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "single point",
			points:       createPoints(5.0),
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "zero volatility",
			points:       createPoints(5.0, 5.0, 5.0),
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "positive sharpe ratio",
			points:       createPoints(1.0, 2.0, 3.0, 4.0, 5.0),
			riskFreeRate: FromFloat64(1.0),
			expected:     FromFloat64(1.41421356),
		},
		{
			name:         "negative sharpe ratio",
			points:       createPoints(1.0, 2.0, 3.0, 4.0, 5.0),
			riskFreeRate: FromFloat64(5.0),
			expected:     FromFloat64(-1.41421356),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SharpeRatio(tt.points, tt.riskFreeRate)
			assertPointEqual(t, tt.expected, result, 0.0001, "SharpeRatio calculation")
		})
	}
}

func TestFixedMath_SortinoRatio(t *testing.T) {
	tests := []struct {
		name         string
		points       []Point
		riskFreeRate Point
		expected     Point
	}{
		{
			name:         "empty slice",
			points:       []Point{},
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "no downside deviation",
			points:       createPoints(1.0, 2.0, 3.0),
			riskFreeRate: Zero,
			expected:     Zero,
		},
		{
			name:         "with downside deviation",
			points:       createPoints(-2.0, -1.0, 1.0, 2.0, 3.0),
			riskFreeRate: Zero,
			expected:     FromFloat64(0.3794733192202055198),
		},
		{
			name:         "all negative returns",
			points:       createPoints(-3.0, -2.0, -1.0),
			riskFreeRate: Zero,
			expected:     FromFloat64(-0.9258200997725514614),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortinoRatio(tt.points, tt.riskFreeRate)
			assertPointEqual(t, tt.expected, result, 0.0001, "SortinoRatio calculation")
		})
	}
}

func BenchmarkFixedMath_Mean(b *testing.B) {
	points := createPoints(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(points)
	}
}

func BenchmarkFixedMath_DownsideDev(b *testing.B) {
	points := createPoints(-2.0, -1.0, 0.0, 1.0, 2.0, 3.0, 4.0, 5.0)
	riskFreeRate := FromFloat64(1.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DownsideDev(points, riskFreeRate)
	}
}

func BenchmarkFixedMath_SampleDownsideDev(b *testing.B) {
	points := createPoints(-2.0, -1.0, 0.0, 1.0, 2.0, 3.0, 4.0, 5.0)
	riskFreeRate := FromFloat64(1.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SampleDownsideDev(points, riskFreeRate)
	}
}

func BenchmarkFixedMath_StdDev(b *testing.B) {
	points := createPoints(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	mean := FromFloat64(5.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StdDev(points, mean)
	}
}

func BenchmarkFixedMath_SampleStdDev(b *testing.B) {
	points := createPoints(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	mean := FromFloat64(5.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SampleStdDev(points, mean)
	}
}

func BenchmarkFixedMath_Variance(b *testing.B) {
	points := createPoints(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	mean := FromFloat64(5.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Variance(points, mean)
	}
}

func BenchmarkFixedMath_SampleVariance(b *testing.B) {
	points := createPoints(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	mean := FromFloat64(5.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SampleVariance(points, mean)
	}
}

func BenchmarkFixedMath_SharpeRatio(b *testing.B) {
	points := createPoints(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	riskFreeRate := FromFloat64(2.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SharpeRatio(points, riskFreeRate)
	}
}

func BenchmarkFixedMath_SortinoRatio(b *testing.B) {
	points := createPoints(-2.0, -1.0, 0.0, 1.0, 2.0, 3.0, 4.0, 5.0)
	riskFreeRate := FromFloat64(1.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SortinoRatio(points, riskFreeRate)
	}
}

// Large dataset benchmarks
func BenchmarkFixedMath_Mean_Large(b *testing.B) {
	points := make([]Point, 1000)
	for i := range points {
		points[i] = FromFloat64(float64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(points)
	}
}

func BenchmarkFixedMath_StdDev_Large(b *testing.B) {
	points := make([]Point, 1000)
	for i := range points {
		points[i] = FromFloat64(float64(i))
	}
	mean := FromFloat64(499.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StdDev(points, mean)
	}
}

func BenchmarkFixedMath_SharpeRatio_Large(b *testing.B) {
	points := make([]Point, 1000)
	for i := range points {
		points[i] = FromFloat64(float64(i%100) / 10.0)
	}
	riskFreeRate := FromFloat64(2.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SharpeRatio(points, riskFreeRate)
	}
}
