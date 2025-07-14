package indicators

import (
	"fmt"
	"testing"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func TestIndicatorsZScore_Value(t *testing.T) {
	tests := []struct {
		name       string
		windowSize int
		data       []float64
		want       float64
		wantReady  bool
	}{
		{
			name:       "not enough data",
			windowSize: 3,
			data:       []float64{1.0, 2.0},
			want:       0.0,
			wantReady:  false,
		},
		{
			name:       "exact window size",
			windowSize: 3,
			data:       []float64{1.0, 2.0, 3.0},
			want:       1,
			wantReady:  true,
		},
		{
			name:       "more than window size",
			windowSize: 3,
			data:       []float64{1.0, 2.0, 3.0, 4.0},
			want:       1,
			wantReady:  true,
		},
		{
			name:       "larger window",
			windowSize: 5,
			data:       []float64{10.0, 12.0, 14.0, 16.0, 18.0},
			want:       1.2649110640673518,
			wantReady:  true,
		},
		{
			name:       "negative values",
			windowSize: 3,
			data:       []float64{-3.0, -2.0, -1.0},
			want:       1,
			wantReady:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := NewZScore(tt.windowSize)

			for _, v := range tt.data {
				z.AddPoint(fixed.FromFloat64(v))
			}

			if got := z.IsReady(); got != tt.wantReady {
				t.Errorf("IsReady() = %v, want %v", got, tt.wantReady)
			}

			if tt.wantReady {
				got := z.Value()
				gotFloat, _ := got.Float64()

				diff := gotFloat - tt.want
				if diff < 0 {
					diff = -diff
				}
				if diff > 0.000001 {
					t.Errorf("Value() = %v, want %v", gotFloat, tt.want)
				}
			}
		})
	}
}

func TestIndicatorsZScore_AddPoint(t *testing.T) {
	z := NewZScore(3)

	for i := 0; i < 10; i++ {
		z.AddPoint(fixed.FromFloat64(float64(i)))
	}

	if !z.IsReady() {
		t.Error("Expected ZScore to be ready after adding more points than window size")
	}
}

func BenchmarkIndicatorsZScore_AddPoint(b *testing.B) {
	z := NewZScore(20)
	p := fixed.FromFloat64(100.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		z.AddPoint(p)
	}
}

func BenchmarkIndicatorsZScore_Value(b *testing.B) {
	z := NewZScore(20)

	for i := 0; i < 20; i++ {
		z.AddPoint(fixed.FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = z.Value()
	}
}

func BenchmarkIndicatorsZScore_IsReady(b *testing.B) {
	z := NewZScore(20)

	for i := 0; i < 20; i++ {
		z.AddPoint(fixed.FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = z.IsReady()
	}
}

func BenchmarkIndicatorsZScore_FullCycle(b *testing.B) {
	windowSizes := []int{10, 50, 100, 500}

	for _, ws := range windowSizes {
		b.Run(fmt.Sprintf("window_%d", ws), func(b *testing.B) {
			z := NewZScore(ws)

			for i := 0; i < b.N; i++ {
				z.AddPoint(fixed.FromFloat64(float64(i % 100)))
				if z.IsReady() {
					_ = z.Value()
				}
			}
		})
	}
}
