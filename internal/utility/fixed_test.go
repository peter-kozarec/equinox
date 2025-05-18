package utility

import (
	"testing"
)

func Test_FixedArithmetic(t *testing.T) {
	a := NewFixed(12345, 2) // 123.45
	b := NewFixed(6789, 2)  // 67.89

	res := a.Add(b)
	expected := NewFixed(19134, 2) // 191.34
	if res != expected {
		t.Errorf("Add failed: got %v, want %v", res, expected)
	}

	res = a.Sub(b)
	expected = NewFixed(5556, 2) // 55.56
	if res != expected {
		t.Errorf("Sub failed: got %v, want %v", res, expected)
	}

	res = a.Mul(b)                   // 123.45 * 67.89
	expected = NewFixed(83810205, 4) // ~8381.0205
	if res != expected {
		t.Errorf("Mul failed: got %v, want %v", res, expected)
	}

	res = a.Div(b)              // 123.45 / 67.89
	expected = NewFixed(181, 2) // ~1.81
	if res.Value/100 != expected.Value/100 {
		t.Errorf("Div failed: got %v, want %v", res, expected)
	}
}

func Test_FixedIntOps(t *testing.T) {
	a := NewFixed(10000, 2) // 100.00

	res := a.AddInt(5)
	expected := NewFixed(10500, 2)
	if res != expected {
		t.Errorf("AddInt failed: got %v, want %v", res, expected)
	}

	res = a.SubInt(30)
	expected = NewFixed(7000, 2)
	if res != expected {
		t.Errorf("SubInt failed: got %v, want %v", res, expected)
	}

	res = a.MulInt(3)
	expected = NewFixed(30000, 2)
	if res != expected {
		t.Errorf("MulInt failed: got %v, want %v", res, expected)
	}

	res = a.DivInt(4)
	expected = NewFixed(2500, 2)
	if res != expected {
		t.Errorf("DivInt failed: got %v, want %v", res, expected)
	}
}

func Test_FixedComparison(t *testing.T) {
	a := NewFixed(5000, 2) // 50.00
	b := NewFixed(7500, 2) // 75.00
	c := NewFixed(5000, 2) // 50.00

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
	a := NewFixed(12345, 2)
	expected := "123.45"
	if a.String() != expected {
		t.Errorf("String failed: got %s, want %s", a.String(), expected)
	}
}
