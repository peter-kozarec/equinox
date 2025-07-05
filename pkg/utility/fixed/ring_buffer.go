package fixed

import "fmt"

type RingBuffer struct {
	buffer   []Point
	capacity int
	size     int
	tail     int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		panic("capacity must be positive")
	}
	return &RingBuffer{
		buffer:   make([]Point, capacity),
		capacity: capacity,
	}
}

func (r *RingBuffer) Size() int {
	return r.size
}

func (r *RingBuffer) Capacity() int {
	return r.capacity
}

func (r *RingBuffer) IsEmpty() bool {
	return r.size == 0
}

func (r *RingBuffer) IsFull() bool {
	return r.size == r.capacity
}

func (r *RingBuffer) Clear() {
	r.size = 0
	r.tail = 0
}

func (r *RingBuffer) Add(p Point) {
	r.buffer[r.tail] = p
	r.tail = (r.tail + 1) % r.capacity

	if r.size < r.capacity {
		r.size++
	}
}

func (r *RingBuffer) Get(idx int) Point {
	if idx < 0 || idx >= r.size {
		panic(fmt.Sprintf("index %d out of range [0, %d)", idx, r.size))
	}

	if r.size < r.capacity {
		actualIdx := r.tail - 1 - idx
		if actualIdx < 0 {
			actualIdx += r.capacity
		}
		return r.buffer[actualIdx]
	}

	actualIdx := (r.tail - 1 - idx + r.capacity) % r.capacity
	return r.buffer[actualIdx]
}

func (r *RingBuffer) Latest() Point {
	if r.size == 0 {
		panic("buffer is empty")
	}
	return r.Get(0)
}

func (r *RingBuffer) Oldest() Point {
	if r.size == 0 {
		panic("buffer is empty")
	}
	return r.Get(r.size - 1)
}

func (r *RingBuffer) ToSliceLifo() []Point {
	if r.size == 0 {
		return nil
	}

	result := make([]Point, r.size)
	for i := 0; i < r.size; i++ {
		result[i] = r.Get(i)
	}
	return result
}

func (r *RingBuffer) ToSliceFifo() []Point {
	if r.size == 0 {
		return nil
	}

	result := make([]Point, r.size)
	for i := 0; i < r.size; i++ {
		result[i] = r.Get(r.size - 1 - i)
	}
	return result
}

func (r *RingBuffer) ForEachLifo(f func(Point)) {
	for i := 0; i < r.size; i++ {
		f(r.Get(i))
	}
}

func (r *RingBuffer) ForEachFifo(f func(Point)) {
	for i := r.size - 1; i >= 0; i-- {
		f(r.Get(i))
	}
}

func (r *RingBuffer) Sum() Point {
	sum := Zero
	r.ForEachLifo(func(p Point) {
		sum = sum.Add(p)
	})
	return sum
}

func (r *RingBuffer) Mean() Point {
	if r.size == 0 {
		return Zero
	}
	return r.Sum().DivInt(r.size)
}

func (r *RingBuffer) StdDev() Point {
	if r.size <= 1 {
		return Zero
	}

	mean := r.Mean()
	sumSquaredDiff := Zero

	r.ForEachFifo(func(p Point) {
		diff := p.Sub(mean)
		sumSquaredDiff = sumSquaredDiff.Add(diff.Mul(diff))
	})

	return sumSquaredDiff.DivInt(r.size).Sqrt()
}

func (r *RingBuffer) SampleStdDev() Point {
	if r.size <= 1 {
		return Zero
	}

	mean := r.Mean()
	sumSquaredDiff := Zero

	r.ForEachFifo(func(p Point) {
		diff := p.Sub(mean)
		sumSquaredDiff = sumSquaredDiff.Add(diff.Mul(diff))
	})

	return sumSquaredDiff.DivInt(r.size - 1).Sqrt()
}

func (r *RingBuffer) Variance() Point {
	if r.size <= 1 {
		return Zero
	}

	mean := r.Mean()
	sumSquaredDiff := Zero

	r.ForEachFifo(func(p Point) {
		diff := p.Sub(mean)
		sumSquaredDiff = sumSquaredDiff.Add(diff.Mul(diff))
	})

	return sumSquaredDiff.DivInt(r.size)
}

func (r *RingBuffer) SampleVariance() Point {
	if r.size <= 1 {
		return Zero
	}

	mean := r.Mean()
	sumSquaredDiff := Zero

	r.ForEachFifo(func(p Point) {
		diff := p.Sub(mean)
		sumSquaredDiff = sumSquaredDiff.Add(diff.Mul(diff))
	})

	return sumSquaredDiff.DivInt(r.size - 1)
}

func (r *RingBuffer) Min() Point {
	if r.size == 0 {
		panic("buffer is empty")
	}

	minVal := r.Get(0)
	for i := 1; i < r.size; i++ {
		val := r.Get(i)
		if val.Lt(minVal) {
			minVal = val
		}
	}
	return minVal
}

func (r *RingBuffer) Max() Point {
	if r.size == 0 {
		panic("buffer is empty")
	}

	maxVal := r.Get(0)
	for i := 1; i < r.size; i++ {
		val := r.Get(i)
		if val.Gt(maxVal) {
			maxVal = val
		}
	}
	return maxVal
}
