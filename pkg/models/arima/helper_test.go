package arima

import (
	"testing"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func TestModel_SolveLinearSystem(t *testing.T) {
	tests := []struct {
		name     string
		A        [][]fixed.Point
		b        []fixed.Point
		expected []fixed.Point
		nilOut   bool
	}{
		{
			name: "2x2 system",
			A: [][]fixed.Point{
				{fixed.FromInt64(2, 0), fixed.FromInt64(1, 0)},
				{fixed.FromInt64(5, 0), fixed.FromInt64(7, 0)},
			},
			b: []fixed.Point{
				fixed.FromInt64(11, 0),
				fixed.FromInt64(13, 0),
			},
			expected: []fixed.Point{
				fixed.FromInt64(7111111, 6),
				fixed.FromInt64(-3222222, 6),
			},
			nilOut: false,
		},
		{
			name: "Singular matrix",
			A: [][]fixed.Point{
				{fixed.FromInt64(1, 0), fixed.FromInt64(2, 0)},
				{fixed.FromInt64(2, 0), fixed.FromInt64(4, 0)},
			},
			b: []fixed.Point{
				fixed.FromInt64(5, 0),
				fixed.FromInt64(10, 0),
			},
			expected: nil,
			nilOut:   true,
		},
		{
			name: "Zero row matrix",
			A: [][]fixed.Point{
				{fixed.FromInt64(1, 0), fixed.FromInt64(2, 0)},
				{fixed.Zero, fixed.Zero},
			},
			b:        []fixed.Point{fixed.FromInt64(3, 0), fixed.Zero},
			expected: nil,
			nilOut:   true,
		},
		{
			name: "3x3 system",
			A: [][]fixed.Point{
				{fixed.FromInt64(2, 0), fixed.FromInt64(1, 0), fixed.FromInt64(-1, 0)},
				{fixed.FromInt64(-3, 0), fixed.FromInt64(-1, 0), fixed.FromInt64(2, 0)},
				{fixed.FromInt64(-2, 0), fixed.FromInt64(1, 0), fixed.FromInt64(2, 0)},
			},
			b: []fixed.Point{
				fixed.FromInt64(8, 0),
				fixed.FromInt64(-11, 0),
				fixed.FromInt64(-3, 0),
			},
			expected: []fixed.Point{
				fixed.FromInt64(2, 0),
				fixed.FromInt64(3, 0),
				fixed.FromInt64(-1, 0),
			},
			nilOut: false,
		},
		{
			name: "Identity matrix",
			A: [][]fixed.Point{
				{fixed.One, fixed.Zero},
				{fixed.Zero, fixed.One},
			},
			b: []fixed.Point{fixed.FromInt64(7, 0), fixed.FromInt64(-3, 0)},
			expected: []fixed.Point{
				fixed.FromInt64(7, 0),
				fixed.FromInt64(-3, 0),
			},
			nilOut: false,
		},
	}

	tolerance := fixed.FromFloat64(1e-6)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := solveLinearSystem(tt.A, tt.b)

			if tt.nilOut {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("unexpected nil output")
			}

			if len(got) != len(tt.expected) {
				t.Fatalf("length mismatch: got %d, want %d", len(got), len(tt.expected))
			}

			for i := range got {
				diff := got[i].Sub(tt.expected[i]).Abs()
				if diff.Gt(tolerance) {
					t.Errorf("x[%d] = %s; want %s (diff = %s)", i, got[i], tt.expected[i], diff)
				}
			}
		})
	}
}

func TestModel_SolveNormalEquations(t *testing.T) {
	tests := []struct {
		name string
		X    [][]fixed.Point
		y    []fixed.Point
		want []fixed.Point
	}{
		{
			name: "Linear model y=2x+1",
			X: [][]fixed.Point{
				{fixed.One, fixed.FromInt64(1, 0)},
				{fixed.One, fixed.FromInt64(2, 0)},
				{fixed.One, fixed.FromInt64(3, 0)},
			},
			y: []fixed.Point{
				fixed.FromInt64(3, 0),
				fixed.FromInt64(5, 0),
				fixed.FromInt64(7, 0),
			},
			want: []fixed.Point{fixed.FromInt64(1, 0), fixed.FromInt64(2, 0)},
		},
		{
			name: "Quadratic model y=1+2x+3x^2",
			X: [][]fixed.Point{
				{fixed.One, fixed.FromInt64(0, 0), fixed.FromInt64(0, 0)},
				{fixed.One, fixed.FromInt64(1, 0), fixed.FromInt64(1, 0)},
				{fixed.One, fixed.FromInt64(2, 0), fixed.FromInt64(4, 0)},
			},
			y: []fixed.Point{
				fixed.FromInt64(1, 0),
				fixed.FromInt64(6, 0),
				fixed.FromInt64(17, 0),
			},
			want: []fixed.Point{fixed.FromInt64(1, 0), fixed.FromInt64(2, 0), fixed.FromInt64(3, 0)},
		},
		{
			name: "Constant model y=5",
			X: [][]fixed.Point{
				{fixed.One},
				{fixed.One},
				{fixed.One},
			},
			y: []fixed.Point{
				fixed.FromInt64(5, 0),
				fixed.FromInt64(5, 0),
				fixed.FromInt64(5, 0),
			},
			want: []fixed.Point{fixed.FromInt64(5, 0)},
		},
	}

	tolerance := fixed.FromInt64(1, 6)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := solveNormalEquations(tt.X, tt.y)

			if len(got) != len(tt.want) {
				t.Fatalf("length mismatch: got %d, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if got[i].Sub(tt.want[i]).Abs().Gt(tolerance) {
					t.Errorf("beta[%d] = %s; want %s", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestModel_BinomialCoefficient(t *testing.T) {
	tests := []struct {
		n, k     int
		expected fixed.Point
	}{
		{5, 0, fixed.One},
		{5, 1, fixed.FromInt64(5, 0)},
		{5, 2, fixed.FromInt64(10, 0)},
		{5, 3, fixed.FromInt64(10, 0)},
		{5, 4, fixed.FromInt64(5, 0)},
		{5, 5, fixed.One},
		{6, 2, fixed.FromInt64(15, 0)},
		{0, 1, fixed.Zero},
		{20, 10, fixed.FromInt64(184756, 0)},
		{3, 5, fixed.Zero},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := binomialCoefficient(tt.n, tt.k)
			if !got.Eq(tt.expected) {
				t.Errorf("binomialCoefficient(%d, %d) = %s, want %s", tt.n, tt.k, got, tt.expected)
			}
		})
	}
}
