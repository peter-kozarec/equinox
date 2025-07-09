package utility

import (
	"math"
	"testing"
)

func TestUtilityConversion_I32ToU32(t *testing.T) {
	tests := []struct {
		input    int32
		expected uint32
		hasError bool
	}{
		{0, 0, false},
		{1, 1, false},
		{math.MaxInt32, uint32(math.MaxInt32), false},
		{-1, 0, true},
		{-42, 0, true},
	}

	for _, tt := range tests {
		result, err := I32ToU32(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("I32ToU32(%d) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("I32ToU32(%d) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("I32ToU32(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestUtilityConversion_I32ToU32Unsafe(t *testing.T) {
	tests := []struct {
		input       int32
		expected    uint32
		shouldPanic bool
	}{
		{0, 0, false},
		{1, 1, false},
		{math.MaxInt32, uint32(math.MaxInt32), false},
		{-1, 0, true},
		{-100, 0, true},
	}

	for _, tt := range tests {
		if tt.shouldPanic {
			t.Run("panic", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("I32ToU32Unsafe(%d) expected panic, got none", tt.input)
					}
				}()
				I32ToU32Unsafe(tt.input)
			})
		} else {
			result := I32ToU32Unsafe(tt.input)
			if result != tt.expected {
				t.Errorf("I32ToU32Unsafe(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestUtilityConversion_U32ToI32(t *testing.T) {
	tests := []struct {
		input    uint32
		expected int32
		hasError bool
	}{
		{0, 0, false},
		{1, 1, false},
		{math.MaxInt32, math.MaxInt32, false},
		{math.MaxInt32 - 1, math.MaxInt32 - 1, false},
		{uint32(math.MaxInt32) + 1, 0, true},
		{math.MaxUint32, 0, true},
		{math.MaxUint32 - 1, 0, true},
	}

	for _, tt := range tests {
		result, err := U32ToI32(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("U32ToI32(%d) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("U32ToI32(%d) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("U32ToI32(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestUtilityConversion_U32ToI32Unsafe(t *testing.T) {
	tests := []struct {
		input       uint32
		expected    int32
		shouldPanic bool
	}{
		{0, 0, false},
		{1, 1, false},
		{math.MaxInt32, math.MaxInt32, false},
		{math.MaxInt32 - 1, math.MaxInt32 - 1, false},
		{uint32(math.MaxInt32) + 1, 0, true},
		{math.MaxUint32, 0, true},
		{math.MaxUint32 - 1, 0, true},
	}

	for _, tt := range tests {
		if tt.shouldPanic {
			t.Run("panic", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("U32ToI32Unsafe(%d) expected panic, got none", tt.input)
					}
				}()
				U32ToI32Unsafe(tt.input)
			})
		} else {
			result := U32ToI32Unsafe(tt.input)
			if result != tt.expected {
				t.Errorf("U32ToI32Unsafe(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestUtilityConversion_I64ToU64(t *testing.T) {
	tests := []struct {
		input    int64
		expected uint64
		hasError bool
	}{
		{0, 0, false},
		{1, 1, false},
		{math.MaxInt64, uint64(math.MaxInt64), false},
		{math.MaxInt64 - 1, uint64(math.MaxInt64 - 1), false},
		{-1, 0, true},
		{-42, 0, true},
		{-math.MaxInt64, 0, true},
		{math.MinInt64, 0, true},
	}

	for _, tt := range tests {
		result, err := I64ToU64(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("I64ToU64(%d) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("I64ToU64(%d) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("I64ToU64(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

func TestUtilityConversion_I64ToU64Unsafe(t *testing.T) {
	tests := []struct {
		input       int64
		expected    uint64
		shouldPanic bool
	}{
		{0, 0, false},
		{1, 1, false},
		{math.MaxInt64, uint64(math.MaxInt64), false},
		{math.MaxInt64 - 1, uint64(math.MaxInt64 - 1), false},
		{-1, 0, true},
		{-100, 0, true},
		{-math.MaxInt64, 0, true},
		{math.MinInt64, 0, true},
	}

	for _, tt := range tests {
		if tt.shouldPanic {
			t.Run("panic", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("I64ToU64Unsafe(%d) expected panic, got none", tt.input)
					}
				}()
				I64ToU64Unsafe(tt.input)
			})
		} else {
			result := I64ToU64Unsafe(tt.input)
			if result != tt.expected {
				t.Errorf("I64ToU64Unsafe(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	}
}

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
	i32EdgeCases := []struct {
		value         int32
		expectedError bool
	}{
		{0, false},
		{42, false},
		{2147483647, false},
		{-1, true},
		{-2147483648, true},
	}

	for _, tc := range i32EdgeCases {
		_, err := I32ToU32(tc.value)
		if (err != nil) != tc.expectedError {
			t.Errorf("I32ToU32(%d) error = %v, expectedError = %v", tc.value, err, tc.expectedError)
		}
	}

	for _, tc := range i32EdgeCases {
		panicked := didPanic(func() {
			_ = I32ToU32Unsafe(tc.value)
		})
		if panicked != tc.expectedError {
			t.Errorf("I32ToU32Unsafe(%d) panicked = %v, expectedPanic = %v", tc.value, panicked, tc.expectedError)
		}
	}

	i64EdgeCases := []struct {
		value         int64
		expectedError bool
	}{
		{0, false},
		{42, false},
		{9223372036854775807, false},
		{-1, true},
		{-9223372036854775808, true},
	}

	for _, tc := range i64EdgeCases {
		_, err := I64ToU64(tc.value)
		if (err != nil) != tc.expectedError {
			t.Errorf("I64ToU64(%d) error = %v, expectedError = %v", tc.value, err, tc.expectedError)
		}
	}

	for _, tc := range i64EdgeCases {
		panicked := didPanic(func() {
			_ = I64ToU64Unsafe(tc.value)
		})
		if panicked != tc.expectedError {
			t.Errorf("I64ToU64Unsafe(%d) panicked = %v, expectedPanic = %v", tc.value, panicked, tc.expectedError)
		}
	}

	u32EdgeCases := []uint32{
		0,
		0x7FFFFFFF,
		0x80000000,
		0xFFFFFFFF,
	}

	for _, v := range u32EdgeCases {
		_, err := U32ToI32(v)
		expectedError := v >= (1 << 31)
		if (err != nil) != expectedError {
			t.Errorf("U32ToI32(%#x) error = %v, expectedError = %v", v, err, expectedError)
		}
	}

	u32UnsafeEdgeCases := []struct {
		value         uint32
		expectedPanic bool
	}{
		{0, false},
		{0x7FFFFFFF, false},
		{0x80000000, true},
		{0xFFFFFFFF, true},
	}

	for _, tc := range u32UnsafeEdgeCases {
		panicked := didPanic(func() {
			_ = U32ToI32Unsafe(tc.value)
		})
		if panicked != tc.expectedPanic {
			t.Errorf("U32ToI32Unsafe(%#x) panicked = %v, expectedPanic = %v", tc.value, panicked, tc.expectedPanic)
		}
	}

	u64EdgeCases := []uint64{
		0x7FFFFFFFFFFFFFFF,
		0x8000000000000000,
		0x8000000000000001,
		0xFFFFFFFFFFFFFFFF,
	}

	for _, v := range u64EdgeCases {
		_, err := U64ToI64(v)
		expectedError := v >= (1 << 63)
		if (err != nil) != expectedError {
			t.Errorf("U64ToI64(%#x) error = %v, expectedError = %v", v, err, expectedError)
		}
	}

	u64UnsafeEdgeCases := []struct {
		value         uint64
		expectedPanic bool
	}{
		{0x7FFFFFFFFFFFFFFF, false},
		{0x8000000000000000, true},
		{0x8000000000000001, true},
		{0xFFFFFFFFFFFFFFFF, true},
	}

	for _, tc := range u64UnsafeEdgeCases {
		panicked := didPanic(func() {
			_ = U64ToI64Unsafe(tc.value)
		})
		if panicked != tc.expectedPanic {
			t.Errorf("U64ToI64Unsafe(%#x) panicked = %v, expectedPanic = %v", tc.value, panicked, tc.expectedPanic)
		}
	}
}

func didPanic(f func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	f()
	return false
}

func BenchmarkUtilityConversion_I32ToU32_Safe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = I32ToU32(int32(i))
	}
}

func BenchmarkUtilityConversion_I32ToU32_Overflow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = I32ToU32(-1)
	}
}

func BenchmarkUtilityConversion_I32ToU32Unsafe_Safe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = I32ToU32Unsafe(int32(i))
	}
}

func BenchmarkUtilityConversion_I32ToU32Unsafe_Boundary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = I32ToU32Unsafe(0x7FFFFFFF)
	}
}

func BenchmarkUtilityConversion_MixedInputs_I32(b *testing.B) {
	inputs := []int32{
		0,
		42,
		100,
		0x7FFFFFFF,
	}

	b.Run("Safe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = I32ToU32(inputs[i%len(inputs)])
		}
	})

	b.Run("Unsafe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = I32ToU32Unsafe(inputs[i%len(inputs)])
		}
	})
}

func BenchmarkUtilityConversion_I64ToU64_Safe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = I64ToU64(int64(i))
	}
}

func BenchmarkUtilityConversion_I64ToU64_Overflow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = I64ToU64(-1)
	}
}

func BenchmarkUtilityConversion_I64ToU64Unsafe_Safe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = I64ToU64Unsafe(int64(i))
	}
}

func BenchmarkUtilityConversion_I64ToU64Unsafe_Boundary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = I64ToU64Unsafe(0x7FFFFFFFFFFFFFFF)
	}
}

func BenchmarkUtilityConversion_MixedInputs_I64(b *testing.B) {
	inputs := []int64{
		0,
		42,
		100,
		0x7FFFFFFFFFFFFFFF,
	}

	b.Run("Safe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = I64ToU64(inputs[i%len(inputs)])
		}
	})

	b.Run("Unsafe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = I64ToU64Unsafe(inputs[i%len(inputs)])
		}
	})
}

func BenchmarkUtilityConversion_U32ToI32_Safe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = U32ToI32(uint32(i))
	}
}

func BenchmarkUtilityConversion_U32ToI32_Overflow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = U32ToI32(0xFFFFFFFF)
	}
}

func BenchmarkUtilityConversion_U32ToI32_Boundary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = U32ToI32(0x7FFFFFFF)
	}
}

func BenchmarkUtilityConversion_U32ToI32Unsafe_Safe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = U32ToI32Unsafe(uint32(i))
	}
}

func BenchmarkUtilityConversion_U32ToI32Unsafe_Boundary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = U32ToI32Unsafe(0x7FFFFFFF)
	}
}

func BenchmarkUtilityConversion_MixedInputs_U32(b *testing.B) {
	mixedInputs := []uint32{
		0,
		42,
		0x7FFFFFFE,
		0x7FFFFFFF,
		0xFFFFFFFF,
	}

	b.Run("Safe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = U32ToI32(mixedInputs[i%len(mixedInputs)])
		}
	})

	safeInputs := []uint32{
		0,
		42,
		0x7FFFFFFE,
		0x7FFFFFFF,
	}

	b.Run("Unsafe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = U32ToI32Unsafe(safeInputs[i%len(safeInputs)])
		}
	})
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

func BenchmarkUtilityConversion_MixedInputs_U64(b *testing.B) {
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
