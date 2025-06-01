package fixed

import (
	"math"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

// Test constants
func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Point
		expected string
	}{
		{"Zero", Zero, "0"},
		{"One", One, "1"},
		{"Sqrt252", Sqrt252, "15.874507866387540"}, // approximately
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Sqrt252" {
				// For Sqrt252, check that it's approximately correct
				if !tt.constant.Gt(New(15, 0)) || !tt.constant.Lt(New(16, 0)) {
					t.Errorf("%s = %s, expected to be between 15 and 16", tt.name, tt.constant.String())
				}
			} else {
				if tt.constant.String() != tt.expected {
					t.Errorf("%s = %s, expected %s", tt.name, tt.constant.String(), tt.expected)
				}
			}
		})
	}
}

// Test constructors
func TestNew(t *testing.T) {
	tests := []struct {
		value     int64
		precision int
		expected  string
	}{
		{123, 0, "123"},
		{123, 2, "1.23"},
		{0, 0, "0"},
		{-456, 1, "-45.6"},
		{1000000, 6, "1.000000"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := New(tt.value, tt.precision)
			if result.String() != tt.expected {
				t.Errorf("New(%d, %d) = %s, expected %s", tt.value, tt.precision, result.String(), tt.expected)
			}
		})
	}
}

func TestFromUint(t *testing.T) {
	tests := []struct {
		value     uint64
		precision int
		expected  string
	}{
		{123, 0, "123"},
		{456, 2, "4.56"},
		{0, 0, "0"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := FromUint(tt.value, tt.precision)
			if result.String() != tt.expected {
				t.Errorf("FromUint(%d, %d) = %s, expected %s", tt.value, tt.precision, result.String(), tt.expected)
			}
		})
	}
}

func TestFromFloat(t *testing.T) {
	tests := []struct {
		value    float64
		expected string
	}{
		{123.45, "123.45"},
		{0.0, "0"},
		{-789.123, "-789.123"},
		{1.0, "1"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := FromFloat(tt.value)
			if result.String() != tt.expected {
				t.Errorf("FromFloat(%f) = %s, expected %s", tt.value, result.String(), tt.expected)
			}
		})
	}
}

// Test arithmetic operations
func TestAdd(t *testing.T) {
	tests := []struct {
		a, b     Point
		expected string
	}{
		{New(100, 0), New(50, 0), "150"},
		{New(125, 2), New(75, 2), "2.00"},
		{New(-50, 0), New(25, 0), "-25"},
		{Zero, New(100, 0), "100"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.a.Add(tt.b)
			if result.String() != tt.expected {
				t.Errorf("%s + %s = %s, expected %s", tt.a.String(), tt.b.String(), result.String(), tt.expected)
			}
		})
	}
}

func TestSub(t *testing.T) {
	tests := []struct {
		a, b     Point
		expected string
	}{
		{New(100, 0), New(50, 0), "50"},
		{New(200, 2), New(100, 2), "1.00"},
		{New(25, 0), New(50, 0), "-25"},
		{Zero, New(100, 0), "-100"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.a.Sub(tt.b)
			if result.String() != tt.expected {
				t.Errorf("%s - %s = %s, expected %s", tt.a.String(), tt.b.String(), result.String(), tt.expected)
			}
		})
	}
}

func TestMul(t *testing.T) {
	tests := []struct {
		a, b     Point
		expected string
	}{
		{New(10, 0), New(5, 0), "50"},
		{New(250, 2), New(400, 2), "10.0000"},
		{New(-3, 0), New(4, 0), "-12"},
		{Zero, New(100, 0), "0"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.a.Mul(tt.b)
			if result.String() != tt.expected {
				t.Errorf("%s * %s = %s, expected %s", tt.a.String(), tt.b.String(), result.String(), tt.expected)
			}
		})
	}
}

func TestDiv(t *testing.T) {
	tests := []struct {
		a, b     Point
		expected string
	}{
		{New(100, 0), New(5, 0), "20"},
		{New(300, 2), New(200, 2), "1.5"},
		{New(-12, 0), New(3, 0), "-4"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.a.Div(tt.b)
			if result.String() != tt.expected {
				t.Errorf("%s / %s = %s, expected %s", tt.a.String(), tt.b.String(), result.String(), tt.expected)
			}
		})
	}
}

func TestDivByZero(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when dividing by zero")
		}
	}()

	New(100, 0).Div(Zero)
}

// Test integer arithmetic operations
func TestIntegerArithmetic(t *testing.T) {
	p := New(100, 0)

	tests := []struct {
		name     string
		result   Point
		expected string
	}{
		{"AddInt64", p.AddInt64(50), "150"},
		{"AddInt", p.AddInt(25), "125"},
		{"SubInt64", p.SubInt64(30), "70"},
		{"SubInt", p.SubInt(40), "60"},
		{"MulInt64", p.MulInt64(2), "200"},
		{"MulInt", p.MulInt(3), "300"},
		{"DivInt64", p.DivInt64(4), "25"},
		{"DivInt", p.DivInt(5), "20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.String() != tt.expected {
				t.Errorf("%s = %s, expected %s", tt.name, tt.result.String(), tt.expected)
			}
		})
	}
}

// Test comparison operations
func TestComparisons(t *testing.T) {
	a := New(100, 0)
	b := New(200, 0)
	c := New(100, 0)

	tests := []struct {
		name     string
		result   bool
		expected bool
	}{
		{"a == c", a.Eq(c), true},
		{"a == b", a.Eq(b), false},
		{"a > b", a.Gt(b), false},
		{"b > a", b.Gt(a), true},
		{"a < b", a.Lt(b), true},
		{"b < a", b.Lt(a), false},
		{"a >= c", a.Gte(c), true},
		{"a >= b", a.Gte(b), false},
		{"b >= a", b.Gte(a), true},
		{"a <= c", a.Lte(c), true},
		{"a <= b", a.Lte(b), true},
		{"b <= a", b.Lte(a), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result != tt.expected {
				t.Errorf("%s = %t, expected %t", tt.name, tt.result, tt.expected)
			}
		})
	}
}

// Test utility methods
func TestAbs(t *testing.T) {
	tests := []struct {
		input    Point
		expected string
	}{
		{New(100, 0), "100"},
		{New(-100, 0), "100"},
		{Zero, "0"},
		{New(-456, 2), "4.56"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.input.Abs()
			if result.String() != tt.expected {
				t.Errorf("Abs(%s) = %s, expected %s", tt.input.String(), result.String(), tt.expected)
			}
		})
	}
}

func TestNeg(t *testing.T) {
	tests := []struct {
		input    Point
		expected string
	}{
		{New(100, 0), "-100"},
		{New(-100, 0), "100"},
		{Zero, "0"},
		{New(456, 2), "-4.56"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.input.Neg()
			if result.String() != tt.expected {
				t.Errorf("Neg(%s) = %s, expected %s", tt.input.String(), result.String(), tt.expected)
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		input    Point
		expected bool
	}{
		{Zero, true},
		{New(0, 5), true},
		{New(1, 0), false},
		{New(-1, 0), false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.input.IsZero()
			if result != tt.expected {
				t.Errorf("IsZero(%s) = %t, expected %t", tt.input.String(), result, tt.expected)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input    Point
		expected string
	}{
		{New(12345, 2), "123.45"},
		{Zero, "0"},
		{New(-6789, 3), "-6.789"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.input.String()
			if result != tt.expected {
				t.Errorf("String() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestPrecision(t *testing.T) {
	tests := []struct {
		input    Point
		expected int
	}{
		{New(123, 0), 3},
		{New(12345, 2), 5},
		{New(123456, 4), 6},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.input.Precision()
			if result != tt.expected {
				t.Errorf("Precision() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

// Test mathematical operations
func TestPow(t *testing.T) {
	tests := []struct {
		base     Point
		exponent Point
		expected string
	}{
		{New(2, 0), New(3, 0), "8"},
		{New(10, 0), New(2, 0), "100"},
		{New(5, 0), New(0, 0), "1"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.base.Pow(tt.exponent)
			if result.String() != tt.expected {
				t.Errorf("Pow(%s, %s) = %s, expected %s", tt.base.String(), tt.exponent.String(), result.String(), tt.expected)
			}
		})
	}
}

func TestSqrt(t *testing.T) {
	tests := []struct {
		input    Point
		expected string
	}{
		{New(4, 0), "2"},
		{New(9, 0), "3"},
		{New(16, 0), "4"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.input.Sqrt()
			if result.String() != tt.expected {
				t.Errorf("Sqrt(%s) = %s, expected %s", tt.input.String(), result.String(), tt.expected)
			}
		})
	}
}

func TestSqrtNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when taking square root of negative number")
		}
	}()

	New(-4, 0).Sqrt()
}

// Test conversion methods
func TestFloat64(t *testing.T) {
	tests := []struct {
		input    Point
		expected float64
	}{
		{New(12345, 2), 123.45},
		{Zero, 0.0},
		{New(-6789, 3), -6.789},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.input.Float64()
			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("Float64() = %f, expected %f", result, tt.expected)
			}
		})
	}
}

func TestRescale(t *testing.T) {
	tests := []struct {
		input    Point
		scale    int
		expected string
	}{
		{New(12345, 2), 3, "123.450"},
		{New(12345, 2), 1, "123.4"},
		{New(12300, 2), 0, "123"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := tt.input.Rescale(tt.scale)
			if result.String() != tt.expected {
				t.Errorf("Rescale(%s, %d) = %s, expected %s", tt.input.String(), tt.scale, result.String(), tt.expected)
			}
		})
	}
}

// Test Zap logging integration
func TestMarshalLogObject(t *testing.T) {
	p := New(12345, 2)

	// Mock encoder to capture the logged value
	encoder := &mockEncoder{fields: make(map[string]interface{})}

	err := p.MarshalLogObject(encoder)
	if err != nil {
		t.Errorf("MarshalLogObject() returned error: %v", err)
	}

	if encoder.fields["decimal"] != "123.45" {
		t.Errorf("MarshalLogObject() logged %v, expected '123.45'", encoder.fields["decimal"])
	}
}

// Mock encoder for testing Zap integration
type mockEncoder struct {
	fields map[string]interface{}
}

func (m *mockEncoder) AddString(key, val string) {
	m.fields[key] = val
}

func (m *mockEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error   { return nil }
func (m *mockEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error { return nil }
func (m *mockEncoder) AddBinary(key string, val []byte)                              {}
func (m *mockEncoder) AddByteString(key string, val []byte)                          {}
func (m *mockEncoder) AddBool(key string, val bool)                                  {}
func (m *mockEncoder) AddComplex128(key string, val complex128)                      {}
func (m *mockEncoder) AddComplex64(key string, val complex64)                        {}
func (m *mockEncoder) AddDuration(key string, val time.Duration)                     {}
func (m *mockEncoder) AddFloat64(key string, val float64)                            {}
func (m *mockEncoder) AddFloat32(key string, val float32)                            {}
func (m *mockEncoder) AddInt(key string, val int)                                    {}
func (m *mockEncoder) AddInt64(key string, val int64)                                {}
func (m *mockEncoder) AddInt32(key string, val int32)                                {}
func (m *mockEncoder) AddInt16(key string, val int16)                                {}
func (m *mockEncoder) AddInt8(key string, val int8)                                  {}
func (m *mockEncoder) AddTime(key string, val time.Time)                             {}
func (m *mockEncoder) AddUint(key string, val uint)                                  {}
func (m *mockEncoder) AddUint64(key string, val uint64)                              {}
func (m *mockEncoder) AddUint32(key string, val uint32)                              {}
func (m *mockEncoder) AddUint16(key string, val uint16)                              {}
func (m *mockEncoder) AddUint8(key string, val uint8)                                {}
func (m *mockEncoder) AddUintptr(key string, val uintptr)                            {}
func (m *mockEncoder) AddReflected(key string, val interface{}) error                { return nil }
func (m *mockEncoder) OpenNamespace(key string)                                      {}

// Test edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("Large numbers", func(t *testing.T) {
		large := New(9223372036854775807, 0) // max int64
		result := large.Add(New(1, 0))
		if result.String() != "9223372036854775808" {
			t.Errorf("Large number addition failed: got %s", result.String())
		}
	})

	t.Run("High precision", func(t *testing.T) {
		highPrec := New(123456789, 8)
		expected := "1.23456789"
		if highPrec.String() != expected {
			t.Errorf("High precision: got %s, expected %s", highPrec.String(), expected)
		}
	})

	t.Run("Zero precision operations", func(t *testing.T) {
		a := New(1, 0)
		b := New(3, 0)
		result := a.Div(b)
		// Should handle division resulting in repeating decimal
		if result.IsZero() {
			t.Errorf("Division by 3 should not result in zero")
		}
	})
}
