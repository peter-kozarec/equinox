package utility

import (
	"testing"
)

func Test_FixedArithmetic(t *testing.T) {
	a := NewFixedFromInt(12345, 2) // 123.45
	b := NewFixedFromInt(6789, 2)  // 67.89

	if res := a.Add(b); !res.Eq(NewFixedFromInt(19134, 2)) {
		t.Errorf("Add failed: got %v", res.String())
	}
	if res := a.Sub(b); !res.Eq(NewFixedFromInt(5556, 2)) {
		t.Errorf("Sub failed: got %v", res.String())
	}
	if res := a.Mul(b); !res.Eq(NewFixedFromInt(83810205, 4)) {
		t.Errorf("Mul failed: got %v", res.String())
	}
}

func Test_FixedIntOps(t *testing.T) {
	a := NewFixedFromInt(10000, 2)

	if res := a.AddInt64(5); !res.Eq(NewFixedFromInt(10500, 2)) {
		t.Errorf("AddInt64 failed")
	}
	if res := a.SubInt64(30); !res.Eq(NewFixedFromInt(7000, 2)) {
		t.Errorf("SubInt64 failed")
	}
	if res := a.MulInt64(3); !res.Eq(NewFixedFromInt(30000, 2)) {
		t.Errorf("MulInt64 failed")
	}
	if res := a.DivInt(4); !res.Eq(NewFixedFromInt(2500, 2)) {
		t.Errorf("DivInt failed")
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

func Benchmark_FixedAdd(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	c := NewFixedFromInt(8765432, 4)
	for i := 0; i < b.N; i++ {
		_ = a.Add(c)
	}
}

func Benchmark_FixedSub(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	c := NewFixedFromInt(8765432, 4)
	for i := 0; i < b.N; i++ {
		_ = a.Sub(c)
	}
}

func Benchmark_FixedMul(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	c := NewFixedFromInt(10000, 2)
	for i := 0; i < b.N; i++ {
		_ = a.Mul(c)
	}
}

func Benchmark_FixedDiv(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	c := NewFixedFromInt(10000, 2)
	for i := 0; i < b.N; i++ {
		_ = a.Div(c)
	}
}

func Benchmark_FixedMulInt(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.MulInt64(3)
	}
}

func Benchmark_FixedDivInt(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.DivInt(3)
	}
}

func Benchmark_FixedAddInt(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.AddInt64(100)
	}
}

func Benchmark_FixedSubInt(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.SubInt64(100)
	}
}

func Benchmark_FixedString(b *testing.B) {
	a := NewFixedFromInt(12345678, 4)
	for i := 0; i < b.N; i++ {
		_ = a.String()
	}
}
