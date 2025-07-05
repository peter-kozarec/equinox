package utility

import (
	"math"
	"testing"
)

func TestUtilityConversion_U64ToI64(t *testing.T) {
	tests := []struct {
		input    uint64
		expected int64
		hasError bool
	}{
		{0, 0, false},
		{1, 1, false},
		{math.MaxInt64, math.MaxInt64, false},
		{math.MaxInt64 - 1, math.MaxInt64 - 1, false},
		{uint64(math.MaxInt64) + 1, 0, true},
		{math.MaxUint64, 0, true},
		{math.MaxUint64 - 1, 0, true},
		{1 << 62, 1 << 62, false},
		{1<<63 - 1, 1<<63 - 1, false},
		{1 << 63, 0, true},
	}

	for _, tt := range tests {
		result, err := U64ToI64(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("U64ToI64(%d) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("U64ToI64(%d) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("U64ToI64(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestUtilityConversion_U64ToI64Unsafe(t *testing.T) {
	tests := []struct {
		input       uint64
		expected    int64
		shouldPanic bool
	}{
		{0, 0, false},
		{1, 1, false},
		{math.MaxInt64, math.MaxInt64, false},
		{math.MaxInt64 - 1, math.MaxInt64 - 1, false},
		{uint64(math.MaxInt64) + 1, 0, true},
		{math.MaxUint64, 0, true},
		{math.MaxUint64 - 1, 0, true},
		{1 << 62, 1 << 62, false},
		{1<<63 - 1, 1<<63 - 1, false},
		{1 << 63, 0, true},
	}

	for _, tt := range tests {
		if tt.shouldPanic {
			t.Run("panic", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("U64ToI64Unsafe(%d) expected panic, got none", tt.input)
					}
				}()
				U64ToI64Unsafe(tt.input)
			})
		} else {
			result := U64ToI64Unsafe(tt.input)
			if result != tt.expected {
				t.Errorf("U64ToI64Unsafe(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestUtilityConversion_EdgeCases(t *testing.T) {
	edgeCases := []uint64{
		0x7FFFFFFFFFFFFFFF,
		0x8000000000000000,
		0x8000000000000001,
		0xFFFFFFFFFFFFFFFF,
	}

	for _, v := range edgeCases {
		_, err := U64ToI64(v)
		expectedError := v >= (1 << 63)
		if (err != nil) != expectedError {
			t.Errorf("U64ToI64(%#x) error = %v, expectedError = %v", v, err, expectedError)
		}
	}
}

func BenchmarkUtilityConversion_U64ToI64_Safe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = U64ToI64(uint64(i))
	}
}

func BenchmarkUtilityConversion_U64ToI64_Overflow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = U64ToI64(math.MaxUint64)
	}
}

func BenchmarkUtilityConversion_U64ToI64_Boundary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = U64ToI64(math.MaxInt64)
	}
}

func BenchmarkUtilityConversion_U64ToI64Unsafe_Safe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = U64ToI64Unsafe(uint64(i))
	}
}

func BenchmarkUtilityConversion_U64ToI64Unsafe_Boundary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = U64ToI64Unsafe(math.MaxInt64)
	}
}

func BenchmarkUtilityConversion_MixedInputs(b *testing.B) {
	inputs := []uint64{
		0,
		42,
		math.MaxInt64 - 1,
		math.MaxInt64,
	}

	b.Run("Safe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = U64ToI64(inputs[i%len(inputs)])
		}
	})

	b.Run("Unsafe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = U64ToI64Unsafe(inputs[i%len(inputs)])
		}
	})
}
