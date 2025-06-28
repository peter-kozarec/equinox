package arima

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"testing"
)

func Test_SolveLinearSystem(t *testing.T) {
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
				{fixed.New(2, 0), fixed.New(1, 0)},
				{fixed.New(5, 0), fixed.New(7, 0)},
			},
			b: []fixed.Point{
				fixed.New(11, 0),
				fixed.New(13, 0),
			},
			expected: []fixed.Point{
				fixed.New(7111111, 6),
				fixed.New(-3222222, 6),
			},
			nilOut: false,
		},
		{
			name: "Singular matrix",
			A: [][]fixed.Point{
				{fixed.New(1, 0), fixed.New(2, 0)},
				{fixed.New(2, 0), fixed.New(4, 0)}, // Linear dependent
			},
			b: []fixed.Point{
				fixed.New(5, 0),
				fixed.New(10, 0),
			},
			expected: nil,
			nilOut:   true,
		},
	}

	tolerance := fixed.FromFloat(1e-6)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := solveLinearSystem(tt.A, tt.b)

			if tt.nilOut {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
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

func Test_SolveNormalEquations(t *testing.T) {
	// Model: y = 2x + 1
	X := [][]fixed.Point{
		{fixed.One, fixed.New(1, 0)},
		{fixed.One, fixed.New(2, 0)},
		{fixed.One, fixed.New(3, 0)},
	}
	y := []fixed.Point{
		fixed.New(3, 0),
		fixed.New(5, 0),
		fixed.New(7, 0),
	}

	got := solveNormalEquations(X, y)
	want := []fixed.Point{fixed.New(1, 0), fixed.New(2, 0)} // intercept=1, slope=2

	for i := range want {
		if !got[i].Sub(want[i]).Abs().Lte(fixed.New(1, 6)) {
			t.Errorf("beta[%d] = %s; want %s", i, got[i].String(), want[i].String())
		}
	}
}

func Test_BinomialCoefficient(t *testing.T) {
	tests := []struct {
		n, k     uint
		expected fixed.Point
	}{
		{5, 0, fixed.One},
		{5, 1, fixed.New(5, 0)},
		{5, 2, fixed.New(10, 0)},
		{5, 3, fixed.New(10, 0)},
		{5, 4, fixed.New(5, 0)},
		{5, 5, fixed.One},
		{6, 2, fixed.New(15, 0)},
		{0, 1, fixed.Zero},
	}

	for _, tt := range tests {
		got := binomialCoefficient(tt.n, tt.k)
		if !got.Eq(tt.expected) {
			t.Errorf("binomialCoefficient(%d, %d) = %s, want %s", tt.n, tt.k, got.String(), tt.expected.String())
		}
	}
}
