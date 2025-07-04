package fixed

import (
	"testing"
)

func TestFixedConstants_Constants(t *testing.T) {
	tests := []struct {
		name     string
		constant Point
		want     string
		wantInt  int64
		scale    int
	}{
		{"NegTen", NegTen, "-10", -10, 0},
		{"NegNine", NegNine, "-9", -9, 0},
		{"NegEight", NegEight, "-8", -8, 0},
		{"NegSeven", NegSeven, "-7", -7, 0},
		{"NegSix", NegSix, "-6", -6, 0},
		{"NegFive", NegFive, "-5", -5, 0},
		{"NegFour", NegFour, "-4", -4, 0},
		{"NegThree", NegThree, "-3", -3, 0},
		{"NegTwo", NegTwo, "-2", -2, 0},
		{"NegOne", NegOne, "-1", -1, 0},
		{"Zero", Zero, "0", 0, 0},
		{"One", One, "1", 1, 0},
		{"Two", Two, "2", 2, 0},
		{"Three", Three, "3", 3, 0},
		{"Four", Four, "4", 4, 0},
		{"Five", Five, "5", 5, 0},
		{"Six", Six, "6", 6, 0},
		{"Seven", Seven, "7", 7, 0},
		{"Eight", Eight, "8", 8, 0},
		{"Nine", Nine, "9", 9, 0},
		{"Ten", Ten, "10", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.constant.String(); got != tt.want {
				t.Errorf("%s.String() = %s; want %s", tt.name, got, tt.want)
			}

			if float64Val, ok := tt.constant.Float64(); !ok || float64Val != float64(tt.wantInt) {
				t.Errorf("%s.Float64() = %f, %v; want %f, true", tt.name, float64Val, ok, float64(tt.wantInt))
			}
		})
	}
}

func TestFixedConstants_DecimalConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Point
		want     string
		wantF64  float64
	}{
		{"PointOne", PointOne, "0.1", 0.1},
		{"PointTwo", PointTwo, "0.2", 0.2},
		{"PointThree", PointThree, "0.3", 0.3},
		{"PointFour", PointFour, "0.4", 0.4},
		{"PointFive", PointFive, "0.5", 0.5},
		{"PointSix", PointSix, "0.6", 0.6},
		{"PointSeven", PointSeven, "0.7", 0.7},
		{"PointEight", PointEight, "0.8", 0.8},
		{"PointNine", PointNine, "0.9", 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.constant.String(); got != tt.want {
				t.Errorf("%s.String() = %s; want %s", tt.name, got, tt.want)
			}

			if float64Val, ok := tt.constant.Float64(); !ok || float64Val != tt.wantF64 {
				t.Errorf("%s.Float64() = %f, %v; want %f, true", tt.name, float64Val, ok, tt.wantF64)
			}
		})
	}
}

func TestFixedConstants_ConstantArithmetic(t *testing.T) {
	tests := []struct {
		name string
		op   func() Point
		want string
	}{
		{"One + One", func() Point { return One.Add(One) }, "2"},
		{"Two * Three", func() Point { return Two.Mul(Three) }, "6"},
		{"Ten - Five", func() Point { return Ten.Sub(Five) }, "5"},
		{"Eight / Two", func() Point { return Eight.Div(Two) }, "4"},
		{"NegOne + One", func() Point { return NegOne.Add(One) }, "0"},
		{"NegFive * Two", func() Point { return NegFive.Mul(Two) }, "-10"},
		{"Zero + Ten", func() Point { return Zero.Add(Ten) }, "10"},
		{"PointFive + PointFive", func() Point { return PointFive.Add(PointFive) }, "1.0"},
		{"One - PointOne", func() Point { return One.Sub(PointOne) }, "0.9"},
		{"PointTwo * Five", func() Point { return PointTwo.Mul(Five) }, "1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op().String(); got != tt.want {
				t.Errorf("%s = %s; want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestFixedConstants_ConstantComparisons(t *testing.T) {
	tests := []struct {
		name   string
		result bool
		want   bool
	}{
		{"Zero.IsZero()", Zero.IsZero(), true},
		{"One.IsZero()", One.IsZero(), false},
		{"One > Zero", One.Gt(Zero), true},
		{"NegOne < Zero", NegOne.Lt(Zero), true},
		{"Five.Eq(Five)", Five.Eq(Five), true},
		{"Ten > NegTen", Ten.Gt(NegTen), true},
		{"PointOne < One", PointOne.Lt(One), true},
		{"PointNine < One", PointNine.Lt(One), true},
		{"Two.Gte(Two)", Two.Gte(Two), true},
		{"Three.Lte(Four)", Three.Lte(Four), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result; got != tt.want {
				t.Errorf("%s = %v; want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestFixedConstants_ConstantChaining(t *testing.T) {
	result := One.Add(Two).Mul(Three).Sub(Four)
	want := "5"
	if result.String() != want {
		t.Errorf("Chained constant operations = %s; want %s", result.String(), want)
	}

	result2 := PointOne.Add(PointTwo).Add(PointThree).Add(PointFour)
	want2 := "1.0"
	if result2.String() != want2 {
		t.Errorf("Chained decimal operations = %s; want %s", result2.String(), want2)
	}
}

func TestFixedConstants_ConstantModifications(t *testing.T) {
	tests := []struct {
		name string
		op   func() Point
		want string
	}{
		{"Five.Neg()", func() Point { return Five.Neg() }, "-5"},
		{"NegThree.Abs()", func() Point { return NegThree.Abs() }, "3"},
		{"Four.Pow(Two)", func() Point { return Four.Pow(Two) }, "16"},
		{"Nine.Sqrt()", func() Point { return Nine.Sqrt() }, "3"},
		{"Ten.DivInt64(2)", func() Point { return Ten.DivInt64(2) }, "5"},
		{"Three.MulInt64(3)", func() Point { return Three.MulInt64(3) }, "9"},
		{"PointFive.Mul(Two)", func() Point { return PointFive.Mul(Two) }, "1.0"},
		{"One.Rescale(2)", func() Point { return One.Rescale(2) }, "1.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op().String(); got != tt.want {
				t.Errorf("%s = %s; want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestFixedConstants_ConstantsImmutability(t *testing.T) {
	original := Five.String()
	_ = Five.Add(Three)
	if Five.String() != original {
		t.Errorf("Constant Five was modified after Add operation")
	}

	_ = Zero.Add(One).Mul(Ten)
	if !Zero.IsZero() {
		t.Errorf("Constant Zero was modified after operations")
	}

	originalPoint := PointFive.String()
	_ = PointFive.Mul(Two)
	if PointFive.String() != originalPoint {
		t.Errorf("Constant PointFive was modified after Mul operation")
	}
}

func BenchmarkFixedConstants_ConstantAccess(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Five
	}
}

func BenchmarkFixedConstants_ConstantOperation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Three.Add(Seven)
	}
}

func BenchmarkFixedConstants_ConstantComparison(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Five.Gt(Three)
	}
}
