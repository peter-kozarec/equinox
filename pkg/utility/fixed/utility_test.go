package fixed

import (
	"testing"
)

func TestFixedUtility_DownsideDev(t *testing.T) {
	tests := []struct {
		name         string
		points       []Point
		riskFreeRate Point
		want         string
	}{
		{
			name:         "empty slice",
			points:       []Point{},
			riskFreeRate: Zero,
			want:         "0",
		},
		{
			name:         "no downside returns",
			points:       []Point{One, Two, Three, Four, Five},
			riskFreeRate: Zero,
			want:         "0",
		},
		{
			name:         "all downside returns",
			points:       []Point{NegTwo, NegOne, Zero},
			riskFreeRate: One,
			want:         "2.160246899469286744",
		},
		{
			name:         "mixed returns",
			points:       []Point{NegOne, Zero, One, Two, Three},
			riskFreeRate: One,
			want:         "1.581138830084189666",
		},
		{
			name:         "decimal risk-free rate",
			points:       []Point{PointOne, PointTwo, PointThree, PointFour},
			riskFreeRate: PointThree,
			want:         "0.1581138830084189666",
		},
		{
			name:         "higher risk-free rate",
			points:       []Point{One, Two, Three, Four, Five},
			riskFreeRate: Six,
			want:         "3.316624790355399849",
		},
		{
			name:         "single downside return",
			points:       []Point{NegOne, Five, Ten},
			riskFreeRate: Zero,
			want:         "0",
		},
		{
			name:         "exact threshold",
			points:       []Point{One, Two, Three, Three, Four},
			riskFreeRate: Three,
			want:         "1.581138830084189666",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DownsideDev(tt.points, tt.riskFreeRate)
			if got.String() != tt.want {
				t.Errorf("DownsideDev() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedUtility_Mean(t *testing.T) {
	tests := []struct {
		name   string
		points []Point
		want   string
	}{
		{
			name:   "empty slice",
			points: []Point{},
			want:   "",
		},
		{
			name:   "single value",
			points: []Point{Five},
			want:   "5",
		},
		{
			name:   "two values",
			points: []Point{Two, Four},
			want:   "3",
		},
		{
			name:   "multiple integers",
			points: []Point{One, Two, Three, Four, Five},
			want:   "3",
		},
		{
			name:   "with negative values",
			points: []Point{NegTwo, Zero, Two, Four},
			want:   "1",
		},
		{
			name:   "decimal values",
			points: []Point{PointOne, PointTwo, PointThree},
			want:   "0.2",
		},
		{
			name:   "mixed decimals and integers",
			points: []Point{One, Two, PointFive},
			want:   "1.166666666666666667",
		},
		{
			name:   "large dataset",
			points: []Point{Ten, Nine, Eight, Seven, Six, Five, Four, Three, Two, One},
			want:   "5.5",
		},
		{
			name:   "all same values",
			points: []Point{Three, Three, Three, Three},
			want:   "3",
		},
		{
			name:   "negative mean",
			points: []Point{NegTen, NegFive, NegThree},
			want:   "-6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Mean(tt.points)
			if len(tt.points) == 0 {
				if got.v != (Point{}).v {
					t.Errorf("Mean() = %v; want zero value", got)
				}
			} else if got.String() != tt.want {
				t.Errorf("Mean() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedUtility_SharpeRatio(t *testing.T) {
	tests := []struct {
		name         string
		points       []Point
		riskFreeRate Point
		want         string
	}{
		{
			name:         "simple positive returns",
			points:       []Point{One, Two, Three, Four, Five},
			riskFreeRate: Zero,
			want:         "2.121320343559642573",
		},
		{
			name:         "with risk-free rate",
			points:       []Point{Two, Three, Four, Five, Six},
			riskFreeRate: One,
			want:         "2.121320343559642573",
		},
		{
			name:         "negative excess returns",
			points:       []Point{One, Two, Three},
			riskFreeRate: Five,
			want:         "-3.674234614174767147",
		},
		{
			name:         "zero excess returns",
			points:       []Point{Three, Three, Three},
			riskFreeRate: Three,
			want:         "0",
		},
		{
			name:         "decimal values",
			points:       []Point{PointOne, PointTwo, PointThree, PointFour, PointFive},
			riskFreeRate: PointOne,
			want:         "1.414213562373095049",
		},
		{
			name:         "high volatility",
			points:       []Point{One, Ten, One, Ten},
			riskFreeRate: Zero,
			want:         "1.222222222222222222",
		},
		{
			name:         "mixed positive and negative",
			points:       []Point{NegOne, Zero, One, Two, Three},
			riskFreeRate: Zero,
			want:         "0.7071067811865475243",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SharpeRatio(tt.points, tt.riskFreeRate)
			if got.String() != tt.want {
				t.Errorf("SharpeRatio() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedUtility_SortinoRatio(t *testing.T) {
	tests := []struct {
		name         string
		points       []Point
		riskFreeRate Point
		want         string
	}{
		{
			name:         "no downside returns",
			points:       []Point{One, Two, Three, Four, Five},
			riskFreeRate: Zero,
			want:         "0",
		},
		{
			name:         "all downside returns",
			points:       []Point{NegTwo, NegOne, Zero},
			riskFreeRate: One,
			want:         "-0.9258200997725514614",
		},
		{
			name:         "mixed returns",
			points:       []Point{NegOne, Zero, One, Two, Three},
			riskFreeRate: One,
			want:         "0",
		},
		{
			name:         "positive excess with downside",
			points:       []Point{Zero, One, Two, Three, Four},
			riskFreeRate: One,
			want:         "0",
		},
		{
			name:         "decimal values",
			points:       []Point{PointOne, PointTwo, PointThree, PointFour, PointFive},
			riskFreeRate: PointThree,
			want:         "0",
		},
		{
			name:         "high volatility with downside",
			points:       []Point{NegTwo, Zero, Two, Four, Six},
			riskFreeRate: Two,
			want:         "0",
		},
		{
			name:         "single downside observation",
			points:       []Point{NegOne, Five, Ten},
			riskFreeRate: Zero,
			want:         "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortinoRatio(tt.points, tt.riskFreeRate)
			if got.String() != tt.want {
				t.Errorf("SortinoRatio() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedUtility_StdDev(t *testing.T) {
	tests := []struct {
		name   string
		points []Point
		mean   Point
		want   string
	}{
		{
			name:   "empty slice",
			points: []Point{},
			mean:   Zero,
			want:   "",
		},
		{
			name:   "single value",
			points: []Point{Five},
			mean:   Five,
			want:   "0",
		},
		{
			name:   "two identical values",
			points: []Point{Three, Three},
			mean:   Three,
			want:   "0",
		},
		{
			name:   "simple dataset",
			points: []Point{Two, Four, Four, Four, Five, Five, Seven, Nine},
			mean:   Five,
			want:   "2",
		},
		{
			name:   "dataset with known stddev",
			points: []Point{One, Two, Three, Four, Five},
			mean:   Three,
			want:   "1.414213562373095049",
		},
		{
			name:   "negative values",
			points: []Point{NegTwo, NegOne, Zero, One, Two},
			mean:   Zero,
			want:   "1.414213562373095049",
		},
		{
			name:   "decimal values",
			points: []Point{PointOne, PointThree, PointFive},
			mean:   PointThree,
			want:   "0.1632993161855452066",
		},
		{
			name:   "larger variance",
			points: []Point{One, Ten},
			mean:   FromFloat64(5.5),
			want:   "4.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StdDev(tt.points, tt.mean)
			if len(tt.points) == 0 {
				if got.v != (Point{}).v {
					t.Errorf("StdDev() = %v; want zero value", got)
				}
			} else if got.String() != tt.want {
				t.Errorf("StdDev() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedUtility_SharpeRatioPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SharpeRatio with zero volatility should not panic")
		}
	}()

	points := []Point{Three, Three, Three, Three}
	result := SharpeRatio(points, Zero)
	if !result.IsZero() {
		t.Errorf("SharpeRatio with zero volatility should return zero, got %s", result.String())
	}
}

func TestFixedUtility_SortinoRatioPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SortinoRatio with zero downside deviation should not panic")
		}
	}()

	points := []Point{One, Two, Three, Four, Five}
	result := SortinoRatio(points, Zero)
	if !result.IsZero() {
		t.Errorf("SortinoRatio with zero downside deviation should return zero, got %s", result.String())
	}
}

func TestFixedUtility_RiskAdjustedMetricsComparison(t *testing.T) {
	points := []Point{NegOne, Zero, One, Two, Three, Four, Five}
	riskFreeRate := One

	sharpe := SharpeRatio(points, riskFreeRate)
	sortino := SortinoRatio(points, riskFreeRate)

	if !sortino.Gt(sharpe) {
		t.Errorf("Expected Sortino ratio (%s) > Sharpe ratio (%s) for positively skewed returns",
			sortino.String(), sharpe.String())
	}
}

func TestFixedUtility_EdgeCases(t *testing.T) {
	t.Run("single observation", func(t *testing.T) {
		points := []Point{Five}
		riskFreeRate := Two

		mean := Mean(points)
		if mean.String() != "5" {
			t.Errorf("Mean of single observation = %s; want 5", mean.String())
		}

		stddev := StdDev(points, mean)
		if stddev.String() != "0" {
			t.Errorf("StdDev of single observation = %s; want 0", stddev.String())
		}

		downside := DownsideDev(points, riskFreeRate)
		if downside.String() != "0" {
			t.Errorf("DownsideDev of single observation above threshold = %s; want 0", downside.String())
		}
	})

	t.Run("all returns equal to risk-free rate", func(t *testing.T) {
		points := []Point{Three, Three, Three, Three}
		riskFreeRate := Three

		downside := DownsideDev(points, riskFreeRate)
		if downside.String() != "0" {
			t.Errorf("DownsideDev with all returns equal to risk-free rate = %s; want 0", downside.String())
		}
	})

	t.Run("extreme values", func(t *testing.T) {
		points := []Point{FromInt64(-1000, 0), FromInt64(1000, 0)}

		mean := Mean(points)
		if mean.String() != "0" {
			t.Errorf("Mean of extreme values = %s; want 0", mean.String())
		}

		stdDev := StdDev(points, mean)
		if stdDev.String() != "1000" {
			t.Errorf("StdDev of extreme values = %s; want 1000", stdDev.String())
		}
	})
}

func TestFixedUtility_MeanAndStdDevTogether(t *testing.T) {
	datasets := []struct {
		name         string
		points       []Point
		expectedMean string
		expectedStd  string
	}{
		{
			name:         "uniform distribution",
			points:       []Point{One, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten},
			expectedMean: "5.5",
			expectedStd:  "2.87228132326901433",
		},
		{
			name:         "normal-like distribution",
			points:       []Point{Two, Three, Three, Four, Four, Four, Five, Five, Six, Six, Seven},
			expectedMean: "4.454545454545454545",
			expectedStd:  "1.437398936440172423",
		},
		{
			name:         "bimodal distribution",
			points:       []Point{One, One, One, Nine, Nine, Nine},
			expectedMean: "5",
			expectedStd:  "4",
		},
	}

	for _, tt := range datasets {
		t.Run(tt.name, func(t *testing.T) {
			mean := Mean(tt.points)
			if mean.String() != tt.expectedMean {
				t.Errorf("Mean() = %s; want %s", mean.String(), tt.expectedMean)
			}

			stddev := StdDev(tt.points, mean)
			if stddev.String() != tt.expectedStd {
				t.Errorf("StdDev() = %s; want %s", stddev.String(), tt.expectedStd)
			}
		})
	}
}

func TestFixedUtility_StdDevWithWrongMean(t *testing.T) {
	points := []Point{One, Two, Three, Four, Five}
	wrongMean := Ten

	stddev := StdDev(points, wrongMean)
	if stddev.String() == "0" {
		t.Errorf("StdDev with wrong mean should not be zero")
	}
}

func TestFixedUtility_MeanPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Mean panicked with: %v", r)
		}
	}()

	largeSlice := make([]Point, 1000)
	for i := range largeSlice {
		largeSlice[i] = FromInt64(int64(i), 0)
	}
	_ = Mean(largeSlice)
}

func TestFixedUtility_StdDevPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("StdDev panicked with: %v", r)
		}
	}()

	points := []Point{FromInt64(1000000, 0), FromInt64(2000000, 0)}
	mean := FromInt64(1500000, 0)
	_ = StdDev(points, mean)
}

func BenchmarkFixedUtility_Mean(b *testing.B) {
	points := []Point{One, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(points)
	}
}

func BenchmarkFixedUtility_StdDev(b *testing.B) {
	points := []Point{One, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten}
	mean := FromFloat64(5.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StdDev(points, mean)
	}
}

func BenchmarkFixedUtility_MeanLarge(b *testing.B) {
	points := make([]Point, 1000)
	for i := range points {
		points[i] = FromInt64(int64(i), 0)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(points)
	}
}

func BenchmarkFixedUtility_StdDevLarge(b *testing.B) {
	points := make([]Point, 1000)
	for i := range points {
		points[i] = FromInt64(int64(i), 0)
	}
	mean := FromFloat64(499.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StdDev(points, mean)
	}
}

func BenchmarkFixedUtility_DownsideDev(b *testing.B) {
	points := []Point{NegTwo, NegOne, Zero, One, Two, Three, Four, Five}
	riskFreeRate := One
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DownsideDev(points, riskFreeRate)
	}
}

func BenchmarkFixedUtility_SharpeRatio(b *testing.B) {
	points := []Point{One, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten}
	riskFreeRate := Two
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SharpeRatio(points, riskFreeRate)
	}
}

func BenchmarkFixedUtility_SortinoRatio(b *testing.B) {
	points := []Point{NegOne, Zero, One, Two, Three, Four, Five, Six, Seven, Eight}
	riskFreeRate := Two
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SortinoRatio(points, riskFreeRate)
	}
}

func BenchmarkFixedUtility_AllMetrics(b *testing.B) {
	points := []Point{NegTwo, NegOne, Zero, One, Two, Three, Four, Five, Six, Seven}
	riskFreeRate := One
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(points)
		_ = StdDev(points, Mean(points))
		_ = DownsideDev(points, riskFreeRate)
		_ = SharpeRatio(points, riskFreeRate)
		_ = SortinoRatio(points, riskFreeRate)
	}
}
