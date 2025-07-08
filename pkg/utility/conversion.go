package utility

import (
	"errors"
	"math"
)

func I32ToU32(i int32) (uint32, error) {
	if i >= 0 {
		return uint32(i), nil // #nosec G115
	}
	return 0, errors.New("integer overflow")
}

func I32ToU32Unsafe(i int32) uint32 {
	if i >= 0 {
		return uint32(i) // #nosec G115
	}
	panic("integer overflow")
}

func U32ToI32(i uint32) (int32, error) {
	if i <= uint32(math.MaxInt32) {
		return int32(i), nil // #nosec G115
	}
	return 0, errors.New("integer overflow")
}

func U32ToI32Unsafe(i uint32) int32 {
	if i <= uint32(math.MaxInt32) {
		return int32(i) // #nosec G115
	}
	panic("integer overflow")
}

func U64ToI64(i uint64) (int64, error) {
	if i <= uint64(math.MaxInt64) {
		return int64(i), nil // #nosec G115
	}
	return 0, errors.New("integer overflow")
}

func U64ToI64Unsafe(i uint64) int64 {
	if i <= uint64(math.MaxInt64) {
		return int64(i) // #nosec G115
	}
	panic("integer overflow")
}
