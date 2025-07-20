package historical

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

type Source[T any] struct {
	dataSourceName string
	reader         *mmap.ReaderAt
	bufferPool     *sync.Pool
}

func NewSource[T any](dataSourceName string) *Source[T] {
	return &Source[T]{
		dataSourceName: dataSourceName,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				buffer := make([]byte, int(unsafe.Sizeof(*new(T))))
				return &buffer
			},
		},
	}
}

func (s *Source[T]) Open() error {
	var err error
	s.reader, err = mmap.Open(s.dataSourceName)
	if err != nil {
		return fmt.Errorf("unable to open data source %q: %w", s.dataSourceName, err)
	}
	return nil
}

func (s *Source[T]) Close() {
	_ = s.reader.Close()
}

func (s *Source[T]) Read(index int64, data *T) error {
	buffer := s.bufferPool.Get().(*[]byte)
	defer s.bufferPool.Put(buffer)

	offset := index * int64(len(*buffer))

	n, err := s.reader.ReadAt(*buffer, offset)
	if err != nil && err != io.EOF {
		return fmt.Errorf("unable to read: %w", err)
	}
	if n < len(*buffer) {
		return ErrEof
	}

	*data = *(*T)(unsafe.Pointer(&(*buffer)[0])) // #nosec G103
	return nil
}

func (s *Source[T]) EntryCount() (int64, error) {

	var entry T
	entrySize := int64(unsafe.Sizeof(entry))
	if entrySize == 0 {
		return 0, fmt.Errorf("size of T is zero")
	}

	fileInfo, err := os.Stat(s.dataSourceName)
	if err != nil {
		return 0, fmt.Errorf("unable to get data source %q stats: %w", s.dataSourceName, err)
	}

	totalSize := fileInfo.Size()
	if totalSize%entrySize != 0 {
		return 0, fmt.Errorf("file size is not a multiple of entry size")
	}

	return totalSize / entrySize, nil
}
