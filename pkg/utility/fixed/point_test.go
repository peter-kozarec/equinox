package fixed

import (
	"math"
	"testing"
)

func TestFixedPoint_FromInt64(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		scale int
		want  string
	}{
		{"zero", 0, 0, "0"},
		{"positive", 123, 0, "123"},
		{"negative", -456, 0, "-456"},
		{"with scale", 123, 2, "1.23"},
		{"negative with scale", -456, 3, "-0.456"},
		{"large number", 9223372036854775807, 0, "9223372036854775807"},
		{"min int64", -9223372036854775808, 0, "-9223372036854775808"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromInt64(tt.value, tt.scale)
			if got.String() != tt.want {
				t.Errorf("FromInt64(%d, %d) = %s; want %s", tt.value, tt.scale, got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_FromFloat64(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{"zero", 0.0, "0"},
		{"positive", 123.45, "123.45"},
		{"negative", -67.89, "-67.89"},
		{"small decimal", 0.0001, "0.0001"},
		{"large number", 1e10, "10000000000"},
		{"negative small", -0.00123, "-0.00123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromFloat64(tt.value)
			if got.String() != tt.want {
				t.Errorf("FromFloat64(%f) = %s; want %s", tt.value, got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_FromFloat64Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("FromFloat64(NaN) did not panic")
		}
	}()
	FromFloat64(math.NaN())
}

func TestFixedPoint_Float64(t *testing.T) {
	tests := []struct {
		name      string
		point     Point
		wantFloat float64
		wantOk    bool
	}{
		{"integer", FromInt64(123, 0), 123.0, true},
		{"decimal", FromFloat64(123.45), 123.45, true},
		{"negative", FromFloat64(-67.89), -67.89, true},
		{"zero", FromInt64(0, 0), 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFloat, gotOk := tt.point.Float64()
			if gotOk != tt.wantOk {
				t.Errorf("Float64() ok = %v; want %v", gotOk, tt.wantOk)
			}
			if gotOk && gotFloat != tt.wantFloat {
				t.Errorf("Float64() = %f; want %f", gotFloat, tt.wantFloat)
			}
		})
	}
}

func TestFixedPoint_Abs(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		want  string
	}{
		{"positive", FromInt64(123, 0), "123"},
		{"negative", FromInt64(-456, 0), "456"},
		{"zero", FromInt64(0, 0), "0"},
		{"decimal positive", FromFloat64(12.34), "12.34"},
		{"decimal negative", FromFloat64(-56.78), "56.78"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.point.Abs()
			if got.String() != tt.want {
				t.Errorf("Abs() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_Neg(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		want  string
	}{
		{"positive", FromInt64(123, 0), "-123"},
		{"negative", FromInt64(-456, 0), "456"},
		{"zero", FromInt64(0, 0), "0"},
		{"decimal", FromFloat64(12.34), "-12.34"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.point.Neg()
			if got.String() != tt.want {
				t.Errorf("Neg() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_Add(t *testing.T) {
	tests := []struct {
		name string
		a, b Point
		want string
	}{
		{"positive integers", FromInt64(123, 0), FromInt64(456, 0), "579"},
		{"positive and negative", FromInt64(100, 0), FromInt64(-50, 0), "50"},
		{"decimals", FromFloat64(12.34), FromFloat64(56.78), "69.12"},
		{"zero addition", FromInt64(123, 0), FromInt64(0, 0), "123"},
		{"negative sum", FromInt64(-100, 0), FromInt64(-200, 0), "-300"},
		{"different scales", FromInt64(1234, 2), FromInt64(5678, 3), "18.018"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Add(tt.b)
			if got.String() != tt.want {
				t.Errorf("Add() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_Sub(t *testing.T) {
	tests := []struct {
		name string
		a, b Point
		want string
	}{
		{"positive result", FromInt64(456, 0), FromInt64(123, 0), "333"},
		{"negative result", FromInt64(100, 0), FromInt64(200, 0), "-100"},
		{"decimals", FromFloat64(56.78), FromFloat64(12.34), "44.44"},
		{"zero subtraction", FromInt64(123, 0), FromInt64(0, 0), "123"},
		{"same values", FromInt64(100, 0), FromInt64(100, 0), "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Sub(tt.b)
			if got.String() != tt.want {
				t.Errorf("Sub() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_Mul(t *testing.T) {
	tests := []struct {
		name string
		a, b Point
		want string
	}{
		{"positive integers", FromInt64(12, 0), FromInt64(34, 0), "408"},
		{"positive and negative", FromInt64(10, 0), FromInt64(-5, 0), "-50"},
		{"decimals", FromFloat64(1.5), FromFloat64(2.5), "3.75"},
		{"zero multiplication", FromInt64(123, 0), FromInt64(0, 0), "0"},
		{"negative product", FromInt64(-10, 0), FromInt64(-20, 0), "200"},
		{"with scale", FromInt64(150, 2), FromInt64(200, 2), "3.0000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Mul(tt.b)
			if got.String() != tt.want {
				t.Errorf("Mul() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_Div(t *testing.T) {
	tests := []struct {
		name string
		a, b Point
		want string
	}{
		{"exact division", FromInt64(100, 0), FromInt64(5, 0), "20"},
		{"decimal result", FromInt64(10, 0), FromInt64(3, 0), "3.333333333333333333"},
		{"negative division", FromInt64(-100, 0), FromInt64(20, 0), "-5"},
		{"decimal division", FromFloat64(7.5), FromFloat64(2.5), "3"},
		{"one division", FromInt64(123, 0), FromInt64(1, 0), "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Div(tt.b)
			if got.String() != tt.want {
				t.Errorf("Div() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_DivPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Div by zero did not panic")
		}
	}()
	FromInt64(10, 0).Div(FromInt64(0, 0))
}

func TestFixedPoint_MulInt64(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		mul   int64
		want  string
	}{
		{"positive multiplication", FromInt64(10, 0), 5, "50"},
		{"negative multiplication", FromInt64(10, 0), -3, "-30"},
		{"zero multiplication", FromInt64(123, 0), 0, "0"},
		{"decimal multiplication", FromFloat64(2.5), 4, "10.0"},
		{"one multiplication", FromInt64(42, 0), 1, "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.point.MulInt64(tt.mul)
			if got.String() != tt.want {
				t.Errorf("MulInt64(%d) = %s; want %s", tt.mul, got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_DivInt64(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		div   int64
		want  string
	}{
		{"exact division", FromInt64(100, 0), 5, "20"},
		{"decimal result", FromInt64(10, 0), 3, "3.333333333333333333"},
		{"negative division", FromInt64(100, 0), -20, "-5"},
		{"decimal division", FromFloat64(7.5), 3, "2.5"},
		{"one division", FromInt64(42, 0), 1, "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.point.DivInt64(tt.div)
			if got.String() != tt.want {
				t.Errorf("DivInt64(%d) = %s; want %s", tt.div, got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_DivInt64Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DivInt64(0) did not panic")
		}
	}()
	FromInt64(10, 0).DivInt64(0)
}

func TestFixedPoint_Comparisons(t *testing.T) {
	a := FromInt64(10, 0)
	b := FromInt64(20, 0)
	c := FromInt64(10, 0)
	d := FromInt64(-5, 0)

	tests := []struct {
		name   string
		fn     func() bool
		expect bool
	}{
		{"10 == 10", func() bool { return a.Eq(c) }, true},
		{"10 == 20", func() bool { return a.Eq(b) }, false},
		{"10 > 20", func() bool { return a.Gt(b) }, false},
		{"20 > 10", func() bool { return b.Gt(a) }, true},
		{"10 > 10", func() bool { return a.Gt(c) }, false},
		{"10 < 20", func() bool { return a.Lt(b) }, true},
		{"20 < 10", func() bool { return b.Lt(a) }, false},
		{"10 < 10", func() bool { return a.Lt(c) }, false},
		{"10 >= 10", func() bool { return a.Gte(c) }, true},
		{"10 >= 20", func() bool { return a.Gte(b) }, false},
		{"20 >= 10", func() bool { return b.Gte(a) }, true},
		{"10 <= 10", func() bool { return a.Lte(c) }, true},
		{"10 <= 20", func() bool { return a.Lte(b) }, true},
		{"20 <= 10", func() bool { return b.Lte(a) }, false},
		{"-5 < 10", func() bool { return d.Lt(a) }, true},
		{"10 > -5", func() bool { return a.Gt(d) }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); got != tt.expect {
				t.Errorf("got %v; want %v", got, tt.expect)
			}
		})
	}
}

func TestFixedPoint_IsZero(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		want  bool
	}{
		{"zero", FromInt64(0, 0), true},
		{"positive", FromInt64(1, 0), false},
		{"negative", FromInt64(-1, 0), false},
		{"zero with scale", FromInt64(0, 5), true},
		{"small decimal", FromFloat64(0.0001), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.point.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestFixedPoint_Rescale(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		scale int
		want  string
	}{
		{"increase scale", FromInt64(123, 0), 2, "123.00"},
		{"decrease scale", FromInt64(12345, 3), 1, "12.3"},
		{"same scale", FromInt64(123, 2), 2, "1.23"},
		{"zero scale", FromFloat64(123.456), 0, "123"},
		{"negative to positive scale", FromInt64(-123, 0), 2, "-123.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.point.Rescale(tt.scale)
			if got.String() != tt.want {
				t.Errorf("Rescale(%d) = %s; want %s", tt.scale, got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_Pow(t *testing.T) {
	tests := []struct {
		name     string
		base     Point
		exponent Point
		want     string
	}{
		{"2^3", FromInt64(2, 0), FromInt64(3, 0), "8"},
		{"10^2", FromInt64(10, 0), FromInt64(2, 0), "100"},
		{"2^0", FromInt64(2, 0), FromInt64(0, 0), "1"},
		{"5^1", FromInt64(5, 0), FromInt64(1, 0), "5"},
		{"1.5^2", FromFloat64(1.5), FromInt64(2, 0), "2.25"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.base.Pow(tt.exponent)
			if got.String() != tt.want {
				t.Errorf("Pow() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_PowPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Pow with negative base and non-integer exponent did not panic")
		}
	}()
	FromInt64(-2, 0).Pow(FromFloat64(0.5))
}

func TestFixedPoint_Sqrt(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		want  string
	}{
		{"sqrt(4)", FromInt64(4, 0), "2"},
		{"sqrt(9)", FromInt64(9, 0), "3"},
		{"sqrt(2)", FromInt64(2, 0), "1.414213562373095049"},
		{"sqrt(100)", FromInt64(100, 0), "10"},
		{"sqrt(0.25)", FromFloat64(0.25), "0.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.point.Sqrt()
			if got.String() != tt.want {
				t.Errorf("Sqrt() = %s; want %s", got.String(), tt.want)
			}
		})
	}
}

func TestFixedPoint_SqrtPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Sqrt of negative did not panic")
		}
	}()
	FromInt64(-4, 0).Sqrt()
}

func TestFixedPoint_Exp(t *testing.T) {
	tests := []struct {
		name  string
		point Point
	}{
		{"exp(0)", FromInt64(0, 0)},
		{"exp(1)", FromInt64(1, 0)},
		{"exp(2)", FromInt64(2, 0)},
		{"exp(-1)", FromInt64(-1, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.point.Exp()
			if got.String() == "" {
				t.Errorf("Exp() returned empty string")
			}
		})
	}
}

func TestFixedPoint_Log(t *testing.T) {
	tests := []struct {
		name  string
		point Point
	}{
		{"log(1)", FromInt64(1, 0)},
		{"log(2)", FromInt64(2, 0)},
		{"log(10)", FromInt64(10, 0)},
		{"log(100)", FromInt64(100, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.point.Log()
			if got.String() == "" {
				t.Errorf("Log() returned empty string")
			}
		})
	}
}

func TestFixedPoint_LogPanic(t *testing.T) {
	tests := []struct {
		name  string
		point Point
	}{
		{"log(0)", FromInt64(0, 0)},
		{"log(-1)", FromInt64(-1, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Log() did not panic for %s", tt.name)
				}
			}()
			tt.point.Log()
		})
	}
}

func TestFixedPoint_String(t *testing.T) {
	tests := []struct {
		name  string
		point Point
		want  string
	}{
		{"integer", FromInt64(123, 0), "123"},
		{"negative", FromInt64(-456, 0), "-456"},
		{"decimal", FromFloat64(12.34), "12.34"},
		{"with scale", FromInt64(1234, 2), "12.34"},
		{"zero", FromInt64(0, 0), "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.point.String(); got != tt.want {
				t.Errorf("String() = %s; want %s", got, tt.want)
			}
		})
	}
}

func TestFixedPoint_ChainedOperations(t *testing.T) {
	a := FromInt64(10, 0)
	b := FromInt64(5, 0)
	c := FromInt64(2, 0)

	result := a.Add(b).Mul(c).Sub(FromInt64(10, 0))
	want := "20"
	if result.String() != want {
		t.Errorf("Chained operations = %s; want %s", result.String(), want)
	}

	result2 := FromFloat64(100).Sqrt().Pow(FromInt64(2, 0))
	want2 := "100"
	if result2.String() != want2 {
		t.Errorf("Chained sqrt/pow = %s; want %s", result2.String(), want2)
	}
}

func TestFixedPoint_EdgeCases(t *testing.T) {
	t.Run("very small numbers", func(t *testing.T) {
		small := FromFloat64(1e-10)
		result := small.Mul(FromFloat64(1e10))
		if result.String() != "1.0000000000" {
			t.Errorf("Small number multiplication = %s; want 1.0000000000", result.String())
		}
	})

	t.Run("chain with zero", func(t *testing.T) {
		result := FromInt64(100, 0).Mul(FromInt64(0, 0)).Add(FromInt64(50, 0))
		if result.String() != "50" {
			t.Errorf("Chain with zero = %s; want 50", result.String())
		}
	})

	t.Run("negative chain", func(t *testing.T) {
		result := FromInt64(-10, 0).Abs().Neg().Add(FromInt64(5, 0))
		if result.String() != "-5" {
			t.Errorf("Negative chain = %s; want -5", result.String())
		}
	})
}

func BenchmarkFixedPoint_Add(b *testing.B) {
	x := FromInt64(123456789, 0)
	y := FromInt64(987654321, 0)
	for i := 0; i < b.N; i++ {
		_ = x.Add(y)
	}
}

func BenchmarkFixedPoint_Mul(b *testing.B) {
	x := FromInt64(123456, 0)
	y := FromInt64(789012, 0)
	for i := 0; i < b.N; i++ {
		_ = x.Mul(y)
	}
}

func BenchmarkFixedPoint_MulInt64(b *testing.B) {
	x := FromInt64(123456, 0)
	y := int64(789012)
	for i := 0; i < b.N; i++ {
		_ = x.MulInt64(y)
	}
}

func BenchmarkFixedPoint_Div(b *testing.B) {
	x := FromInt64(1000000, 0)
	y := FromInt64(37, 0)
	for i := 0; i < b.N; i++ {
		_ = x.Div(y)
	}
}

func BenchmarkFixedPoint_DivInt64(b *testing.B) {
	x := FromInt64(1000000, 0)
	y := int64(37)
	for i := 0; i < b.N; i++ {
		_ = x.DivInt64(y)
	}
}

func BenchmarkFixedPoint_Chained(b *testing.B) {
	x := FromInt64(100, 0)
	y := FromInt64(50, 0)
	z := FromInt64(25, 0)
	for i := 0; i < b.N; i++ {
		_ = x.Add(y).Mul(z).Sub(x)
	}
}
