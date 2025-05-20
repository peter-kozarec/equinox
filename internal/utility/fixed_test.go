package utility

import (
	"testing"
)

func Test_FixedArithmetic(t *testing.T) {
	a := NewFixedFromInt(12345, 2) // 123.45
	b := NewFixedFromInt(6789, 2)  // 67.89

	expectedAdd := MustNewFixed(19134, 2)
	expectedSub := MustNewFixed(5556, 2)
	expectedMul := MustNewFixed(83810205, 4)

	if res := a.Add(b); !res.Eq(expectedAdd) {
		t.Errorf("Add failed: got %v, want %v", res.String(), expectedAdd.String())
	}
	if res := a.Sub(b); !res.Eq(expectedSub) {
		t.Errorf("Sub failed: got %v, want %v", res.String(), expectedSub.String())
	}
	if res := a.Mul(b); !res.Eq(expectedMul) {
		t.Errorf("Mul failed: got %v, want %v", res.String(), expectedMul.String())
	}
}

func Test_FixedIntOps(t *testing.T) {
	a := NewFixedFromInt(10000, 2) // 100.00

	if res := a.AddInt64(5); !res.Eq(MustNewFixed(10500, 2)) {
		t.Errorf("AddInt64 failed: got %v", res.String())
	}
	if res := a.SubInt64(30); !res.Eq(MustNewFixed(7000, 2)) {
		t.Errorf("SubInt64 failed: got %v", res.String())
	}
	if res := a.MulInt64(3); !res.Eq(MustNewFixed(30000, 2)) {
		t.Errorf("MulInt64 failed: got %v", res.String())
	}
	if res := a.DivInt64(4); !res.Eq(MustNewFixed(2500, 2)) {
		t.Errorf("DivInt64 failed: got %v", res.String())
	}
}

func Test_FixedComparison(t *testing.T) {
	a := NewFixedFromInt(5000, 2)
	b := NewFixedFromInt(7500, 2)
	c := NewFixedFromInt(5000, 2)

	if !a.Lt(b) {
		t.Errorf("Expected a < b")
	}
	if !b.Gt(a) {
		t.Errorf("Expected b > a")
	}
	if !a.Eq(c) {
		t.Errorf("Expected a == c")
	}
	if !a.Lte(c) {
		t.Errorf("Expected a <= c")
	}
	if !b.Gte(a) {
		t.Errorf("Expected b >= a")
	}
}

func Test_FixedString(t *testing.T) {
	a := NewFixedFromInt(12345, 2)
	expected := "123.45"
	if a.String() != expected {
		t.Errorf("String failed: got %s, want %s", a.String(), expected)
	}
}

func Test_FixedSqrt(t *testing.T) {
	tests := []struct {
		input    Fixed
		expected Fixed
	}{
		{MustNewFixed(4, 0), MustNewFixed(2, 0)},
		{MustNewFixed(225, 2), MustNewFixed(150, 2)}, // √2.25 = 1.50
	}

	for _, tt := range tests {
		result := tt.input.Sqrt().Rescale(2)
		if !result.Eq(tt.expected) {
			t.Errorf("Sqrt(%v) = %v, want %v", tt.input.String(), result.String(), tt.expected.String())
		}
	}
}

func Test_FixedPow(t *testing.T) {
	base := MustNewFixed(2, 0)
	exp := MustNewFixed(3, 0)
	expected := MustNewFixed(8, 0)

	result := base.Pow(exp)
	if !result.Eq(expected) {
		t.Errorf("Pow(%v, %v) = %v, want %v", base.String(), exp.String(), result.String(), expected.String())
	}
}

func Test_FixedZeroHandling(t *testing.T) {
	zero := ZeroFixed
	nonZero := MustNewFixed(100, 2)

	if !zero.Add(nonZero).Eq(nonZero) {
		t.Errorf("Zero add failed")
	}
	if !nonZero.Sub(zero).Eq(nonZero) {
		t.Errorf("Zero sub failed")
	}
	if !zero.Mul(nonZero).IsZero() {
		t.Errorf("Zero mul failed")
	}
}

func Test_FixedHighPrecision(t *testing.T) {
	a := MustNewFixed(123456789, 8)
	b := MustNewFixed(987654321, 8)
	_ = a.Mul(b) // Should not panic or lose significant precision
}

func Test_FixedDivByZero(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic on division by zero")
		}
	}()
	_ = MustNewFixed(100, 2).Div(ZeroFixed)
}

func Benchmark_FixedAdd(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	c := MustNewFixed(8765432, 4)
	for i := 0; i < b.N; i++ {
		_ = a.Add(c)
	}
}

func Benchmark_FixedSub(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	c := MustNewFixed(8765432, 4)
	for i := 0; i < b.N; i++ {
		_ = a.Sub(c)
	}
}

func Benchmark_FixedMul(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	c := MustNewFixed(10000, 2)
	for i := 0; i < b.N; i++ {
		_ = a.Mul(c)
	}
}

func Benchmark_FixedDiv(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	c := MustNewFixed(10000, 2)
	for i := 0; i < b.N; i++ {
		_ = a.Div(c)
	}
}

func Benchmark_FixedMulInt(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.MulInt64(3)
	}
}

func Benchmark_FixedDivInt(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.DivInt(3)
	}
}

func Benchmark_FixedAddInt(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.AddInt64(100)
	}
}

func Benchmark_FixedSubInt(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.SubInt64(100)
	}
}

func Benchmark_FixedString(b *testing.B) {
	a := MustNewFixed(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.String()
	}
}
