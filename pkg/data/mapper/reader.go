package mapper

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/exp/mmap"
)

var ErrEof = errors.New("EOF")

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
		return ErrEof
	}

	*data = *(*T)(unsafe.Pointer(&(*buffer)[0])) // Unsafe casting, for performance, T must not be padded
	return nil
}

func (r *Reader[T]) EntryCount() (int64, error) {

	var entry T
	entrySize := int64(unsafe.Sizeof(entry))
	if entrySize == 0 {
		return 0, fmt.Errorf("size of T is zero")
	}

	fileInfo, err := os.Stat(r.dataSourceName)
	if err != nil {
		return 0, fmt.Errorf("unable to get data source %q stats: %w", r.dataSourceName, err)
	}

	totalSize := fileInfo.Size()
	if totalSize%entrySize != 0 {
		return 0, fmt.Errorf("file size is not a multiple of entry size")
	}

	return totalSize / entrySize, nil
}
