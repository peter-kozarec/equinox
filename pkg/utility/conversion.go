package utility

import (
	"errors"
)

func U64ToI64(i uint64) (int64, error) {
	if i&(1<<63) == 0 {
		return int64(i), nil // #nosec G115
	}
	return 0, errors.New("integer overflow")
}

func U64ToI64Unsafe(i uint64) int64 {
	if i&(1<<63) == 0 {
		return int64(i) // #nosec G115
	}
	panic("integer overflow")
}
