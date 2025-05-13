package mapper

import (
	"errors"
	"fmt"
	"golang.org/x/exp/mmap"
	"io"
	"sync"
	"unsafe"
)

var EOF = errors.New("EOF")

type Reader[T any] struct {
	dataSourceName string
	reader         *mmap.ReaderAt
	bufferPool     *sync.Pool
}

func NewReader[T any](dataSourceName string) *Reader[T] {
	return &Reader[T]{
		dataSourceName: dataSourceName,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				buffer := make([]byte, int(unsafe.Sizeof(*new(T))))
				return &buffer
			},
		},
	}
}

func (r *Reader[T]) Open() error {
	var err error
	r.reader, err = mmap.Open(r.dataSourceName)
	if err != nil {
		return fmt.Errorf("unable to open data source %q: %w", r.dataSourceName, err)
	}
	return nil
}

func (r *Reader[T]) Close() {
	_ = r.reader.Close()
}

func (r *Reader[T]) Read(index int64, data *T) error {
	buffer := r.bufferPool.Get().(*[]byte)
	defer r.bufferPool.Put(buffer)

	offset := index * int64(len(*buffer))

	n, err := r.reader.ReadAt(*buffer, offset)
	if err != nil && err != io.EOF {
		return fmt.Errorf("unable to read: %w", err)
	}
	if n < len(*buffer) {
		return EOF
	}

	*data = *(*T)(unsafe.Pointer(&(*buffer)[0])) // Unsafe casting, for performance, T must not be padded
	return nil
}
