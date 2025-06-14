package circular

type Buffer[T any] struct {
	capacity uint

	head uint
	size uint
	data []T
}

func NewBuffer[T any](capacity uint) *Buffer[T] {
	if capacity == 0 {
		panic("capacity must > 0")
	}
	return &Buffer[T]{
		capacity: capacity,
		data:     make([]T, capacity),
	}
}

func (b *Buffer[T]) Capacity() uint {
	return b.capacity
}

func (b *Buffer[T]) Size() uint {
	return b.size
}

func (b *Buffer[T]) Push(value T) {
	b.data[b.head] = value
	b.head = (b.head + 1) % b.capacity
	if b.size < b.capacity {
		b.size++
	}
}

func (b *Buffer[T]) Get(idx uint) T {
	if idx >= b.size {
		panic("index out of range")
	}
	if b.size == b.capacity {
		return b.data[(b.head-1-idx+b.size)%b.capacity]
	}
	return b.data[b.head-1-idx]
}

func (b *Buffer[T]) IsFull() bool {
	return b.size == b.capacity
}
