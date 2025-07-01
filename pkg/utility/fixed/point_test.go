package fixed

import (
	"math"
	"testing"
)

func TestPoint_Constants(t *testing.T) {
	if !NegOne.Eq(New(-1, 0)) {
		t.Errorf("NegOne should equal -1")
	}
	if !Zero.Eq(New(0, 0)) {
		t.Errorf("Zero should equal 0")
	}
	if !One.Eq(New(1, 0)) {
		t.Errorf("One should equal 1")
	}
	if Sqrt252.String() != "15.87450786638754" {
		t.Errorf("Sqrt252 should be approximately sqrt(252)")
	}
}

func TestPoint_New(t *testing.T) {
	p := New(123, 2)
	if p.String() != "1.23" {
		t.Errorf("New(123, 2) should be 1.23, got %s", p.String())
	}
}

func TestPoint_FromUint(t *testing.T) {
	p := FromUint(456, 1)
	if p.String() != "45.6" {
		t.Errorf("FromUint(456, 1) should be 45.6, got %s", p.String())
	}
}

func TestPoint_FromFloat(t *testing.T) {
	p := FromFloat(3.14159)
	if math.Abs(p.Float64()-3.14159) > 1e-10 {
		t.Errorf("FromFloat(3.14159) should preserve value, got %f", p.Float64())
	}
}

func TestPoint_Arithmetic(t *testing.T) {
	a := New(100, 2) // 1.00
	b := New(50, 2)  // 0.50

	// Addition
	result := a.Add(b)
	if result.String() != "1.50" {
		t.Errorf("1.00 + 0.50 should be 1.50, got %s", result.String())
	}

	// Subtraction
	result = a.Sub(b)
	if result.String() != "0.50" {
		t.Errorf("1.00 - 0.50 should be 0.50, got %s", result.String())
	}

	// Multiplication
	result = a.Mul(b)
	if result.String() != "0.5000" {
		t.Errorf("1.00 * 0.50 should be 0.5000, got %s", result.String())
	}

	// Division
	result = a.Div(b)
	if result.String() != "2" {
		t.Errorf("1.00 / 0.50 should be 2, got %s", result.String())
	}
}

func TestPoint_IntegerArithmetic(t *testing.T) {
	p := New(100, 2) // 1.00

	// Test all integer operations
	if !p.AddInt64(1).Eq(New(200, 2)) {
		t.Errorf("AddInt64 failed")
	}
	if !p.AddInt(1).Eq(New(200, 2)) {
		t.Errorf("AddInt failed")
	}
	if !p.SubInt64(1).Eq(New(0, 2)) {
		t.Errorf("SubInt64 failed")
	}
	if !p.SubInt(1).Eq(New(0, 2)) {
		t.Errorf("SubInt failed")
	}
	if !p.MulInt64(2).Eq(New(200, 2)) {
		t.Errorf("MulInt64 failed")
	}
	if !p.MulInt(2).Eq(New(200, 2)) {
		t.Errorf("MulInt failed")
	}
	if !p.DivInt64(2).Eq(New(50, 2)) {
		t.Errorf("DivInt64 failed")
	}
	if !p.DivInt(2).Eq(New(50, 2)) {
		t.Errorf("DivInt failed")
	}
}

func TestPoint_Comparison(t *testing.T) {
	a := New(100, 2) // 1.00
	b := New(50, 2)  // 0.50
	c := New(100, 2) // 1.00

	if !a.Eq(c) {
		t.Errorf("Eq failed")
	}
	if !a.Gt(b) {
		t.Errorf("Gt failed")
	}
	if !b.Lt(a) {
		t.Errorf("Lt failed")
	}
	if !a.Gte(c) {
		t.Errorf("Gte failed")
	}
	if !a.Gte(b) {
		t.Errorf("Gte failed")
	}
	if !b.Lte(a) {
		t.Errorf("Lte failed")
	}
	if !a.Lte(c) {
		t.Errorf("Lte failed")
	}
}

func TestPoint_UtilityFunctions(t *testing.T) {
	neg := New(-100, 2)
	pos := New(100, 2)

	// Test Abs
	if !neg.Abs().Eq(pos) {
		t.Errorf("Abs failed")
	}

	// Test Neg
	if !pos.Neg().Eq(neg) {
		t.Errorf("Neg failed")
	}

	// Test IsZero
	if !Zero.IsZero() {
		t.Errorf("Zero should be zero")
	}
	if pos.IsZero() {
		t.Errorf("Positive number should not be zero")
	}

	// Test Precision
	p := New(123, 3)
	if p.Precision() != 3 {
		t.Errorf("Precision should be 3, got %d", p.Precision())
	}
}

func TestPoint_MathFunctions(t *testing.T) {
	// Test Pow
	base := New(2, 0)
	exp := New(3, 0)
	result := base.Pow(exp)
	if !result.Eq(New(8, 0)) {
		t.Errorf("2^3 should be 8, got %s", result.String())
	}

	// Test Sqrt
	four := New(4, 0)
	result = four.Sqrt()
	if !result.Eq(New(2, 0)) {
		t.Errorf("sqrt(4) should be 2, got %s", result.String())
	}

	// Test Exp and Log (basic functionality)
	one := New(1, 0)
	expResult := one.Exp()
	logResult := expResult.Log()
	if math.Abs(logResult.Float64()-1.0) > 1e-10 {
		t.Errorf("log(exp(1)) should be 1, got %f", logResult.Float64())
	}
}

func TestPoint_Rescale(t *testing.T) {
	p := New(123, 2) // 1.23
	rescaled := p.Rescale(4)
	if rescaled.String() != "1.2300" {
		t.Errorf("Rescale to 4 should be 1.2300, got %s", rescaled.String())
	}
}

func TestPoint_Float64(t *testing.T) {
	p := New(314159, 5) // 3.14159
	f := p.Float64()
	if math.Abs(f-3.14159) > 1e-10 {
		t.Errorf("Float64 should be 3.14159, got %f", f)
	}
}

func TestPoint_ClampPoint(t *testing.T) {
	min := New(0, 0)
	max := New(10, 0)

	// Test value below min
	val := New(-5, 0)
	result := ClampPoint(val, min, max)
	if !result.Eq(min) {
		t.Errorf("Clamp should return min for value below range")
	}

	// Test value above max
	val = New(15, 0)
	result = ClampPoint(val, min, max)
	if !result.Eq(max) {
		t.Errorf("Clamp should return max for value above range")
	}

	// Test value in range
	val = New(5, 0)
	result = ClampPoint(val, min, max)
	if !result.Eq(val) {
		t.Errorf("Clamp should return original value for value in range")
	}
}

func TestPoint_MaxPoint(t *testing.T) {
	points := []Point{
		New(1, 0),
		New(5, 0),
		New(3, 0),
		New(2, 0),
	}

	result := MaxPoint(points...)
	if !result.Eq(New(5, 0)) {
		t.Errorf("MaxPoint should return 5, got %s", result.String())
	}
}

func TestPoint_MinPoint(t *testing.T) {
	points := []Point{
		New(1, 0),
		New(5, 0),
		New(3, 0),
		New(2, 0),
	}

	result := MinPoint(points...)
	if !result.Eq(New(1, 0)) {
		t.Errorf("MinPoint should return 1, got %s", result.String())
	}
}

func TestPoint_MaxPointPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MaxPoint should panic with empty slice")
		}
	}()
	MaxPoint()
}

func TestPoint_MinPointPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MinPoint should panic with empty slice")
		}
	}()
	MinPoint()
}

func TestPoint_EdgeCases(t *testing.T) {
	// Test division by zero should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Division by zero should panic")
		}
	}()
	One.Div(Zero)
}

func BenchmarkPoint_Arithmetic(b *testing.B) {
	a := New(100, 2)
	c := New(50, 2)

	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a.Add(c)
		}
	})

	b.Run("Mul", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a.Mul(c)
		}
	})

	b.Run("Div", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			a.Div(c)
		}
	})
}
