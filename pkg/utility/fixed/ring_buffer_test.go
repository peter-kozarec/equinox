package fixed

import (
	"testing"
)

func assertRingBufferEqual(t *testing.T, rb *RingBuffer, expected []float64, msg string) {
	t.Helper()
	if rb.Size() != len(expected) {
		t.Errorf("%s: size mismatch - got %d, want %d", msg, rb.Size(), len(expected))
		return
	}

	for i, exp := range expected {
		got := rb.Get(i)
		want := FromFloat64(exp)
		if !got.Eq(want) {
			t.Errorf("%s: at index %d - got %v, want %v", msg, i, got, want)
		}
	}
}

func TestFixedRingBuffer_NewRingBuffer(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		wantPanic bool
	}{
		{
			name:      "positive capacity",
			capacity:  10,
			wantPanic: false,
		},
		{
			name:      "capacity of 1",
			capacity:  1,
			wantPanic: false,
		},
		{
			name:      "zero capacity",
			capacity:  0,
			wantPanic: true,
		},
		{
			name:      "negative capacity",
			capacity:  -5,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic for capacity %d", tt.capacity)
					}
				}()
			}

			rb := NewRingBuffer(tt.capacity)

			if !tt.wantPanic {
				if rb.Capacity() != tt.capacity {
					t.Errorf("capacity: got %d, want %d", rb.Capacity(), tt.capacity)
				}
				if rb.Size() != 0 {
					t.Errorf("initial size: got %d, want 0", rb.Size())
				}
				if !rb.IsEmpty() {
					t.Error("new buffer should be empty")
				}
				if rb.IsFull() {
					t.Error("new buffer should not be full")
				}
			}
		})
	}
}

func TestFixedRingBuffer_Add(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(FromFloat64(1.0))
	assertRingBufferEqual(t, rb, []float64{1.0}, "after first add")

	rb.Add(FromFloat64(2.0))
	assertRingBufferEqual(t, rb, []float64{2.0, 1.0}, "after second add")

	rb.Add(FromFloat64(3.0))
	assertRingBufferEqual(t, rb, []float64{3.0, 2.0, 1.0}, "after third add")

	rb.Add(FromFloat64(4.0))
	assertRingBufferEqual(t, rb, []float64{4.0, 3.0, 2.0}, "after wraparound")

	rb.Add(FromFloat64(5.0))
	assertRingBufferEqual(t, rb, []float64{5.0, 4.0, 3.0}, "after second wraparound")
}

func TestFixedRingBuffer_Get(t *testing.T) {
	rb := NewRingBuffer(5)

	t.Run("empty buffer panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when getting from empty buffer")
			}
		}()
		rb.Get(0)
	})

	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for _, v := range values {
		rb.Add(FromFloat64(v))
	}

	tests := []struct {
		idx      int
		expected float64
	}{
		{0, 50.0},
		{1, 40.0},
		{2, 30.0},
		{3, 20.0},
		{4, 10.0},
	}

	for _, tt := range tests {
		t.Run("valid index", func(t *testing.T) {
			got := rb.Get(tt.idx)
			want := FromFloat64(tt.expected)
			if !got.Eq(want) {
				t.Errorf("Get(%d): got %v, want %v", tt.idx, got, want)
			}
		})
	}

	invalidIndices := []int{-1, 5, 100}
	for _, idx := range invalidIndices {
		t.Run("invalid index panic", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for index %d", idx)
				}
			}()
			rb.Get(idx)
		})
	}
}

func TestFixedRingBuffer_IsEmpty(t *testing.T) {
	rb := NewRingBuffer(3)

	if !rb.IsEmpty() {
		t.Error("new buffer should be empty")
	}

	rb.Add(FromFloat64(1.0))
	if rb.IsEmpty() {
		t.Error("buffer with elements should not be empty")
	}

	rb.Clear()
	if !rb.IsEmpty() {
		t.Error("cleared buffer should be empty")
	}
}

func TestFixedRingBuffer_IsFull(t *testing.T) {
	rb := NewRingBuffer(2)

	if rb.IsFull() {
		t.Error("new buffer should not be full")
	}

	rb.Add(FromFloat64(1.0))
	if rb.IsFull() {
		t.Error("partially filled buffer should not be full")
	}

	rb.Add(FromFloat64(2.0))
	if !rb.IsFull() {
		t.Error("buffer at capacity should be full")
	}

	rb.Add(FromFloat64(3.0))
	if !rb.IsFull() {
		t.Error("buffer after wraparound should still be full")
	}
}

func TestFixedRingBuffer_Latest(t *testing.T) {
	rb := NewRingBuffer(3)

	t.Run("empty buffer panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		rb.Latest()
	})

	rb.Add(FromFloat64(1.0))
	if !rb.Latest().Eq(FromFloat64(1.0)) {
		t.Errorf("Latest: got %v, want 1.0", rb.Latest())
	}

	rb.Add(FromFloat64(2.0))
	if !rb.Latest().Eq(FromFloat64(2.0)) {
		t.Errorf("Latest: got %v, want 2.0", rb.Latest())
	}
}

func TestFixedRingBuffer_Oldest(t *testing.T) {
	rb := NewRingBuffer(3)

	t.Run("empty buffer panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		rb.Oldest()
	})

	rb.Add(FromFloat64(1.0))
	if !rb.Oldest().Eq(FromFloat64(1.0)) {
		t.Errorf("Oldest: got %v, want 1.0", rb.Oldest())
	}

	rb.Add(FromFloat64(2.0))
	rb.Add(FromFloat64(3.0))
	if !rb.Oldest().Eq(FromFloat64(1.0)) {
		t.Errorf("Oldest: got %v, want 1.0", rb.Oldest())
	}

	rb.Add(FromFloat64(4.0))
	if !rb.Oldest().Eq(FromFloat64(2.0)) {
		t.Errorf("Oldest after wrap: got %v, want 2.0", rb.Oldest())
	}
}

func TestFixedRingBuffer_Data(t *testing.T) {
	rb := NewRingBuffer(3)

	if data := rb.ToSliceLifo(); data != nil {
		t.Error("ToSliceLifo() should return nil for empty buffer")
	}

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	data := rb.ToSliceLifo()
	expected := []float64{2.0, 1.0}
	for i, v := range expected {
		if !data[i].Eq(FromFloat64(v)) {
			t.Errorf("ToSliceLifo()[%d]: got %v, want %v", i, data[i], v)
		}
	}

	rb.Add(FromFloat64(3.0))
	rb.Add(FromFloat64(4.0))
	data = rb.ToSliceLifo()
	expected = []float64{4.0, 3.0, 2.0}
	for i, v := range expected {
		if !data[i].Eq(FromFloat64(v)) {
			t.Errorf("ToSliceLifo()[%d]: got %v, want %v", i, data[i], v)
		}
	}
}

func TestFixedRingBuffer_DataReversed(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	rb.Add(FromFloat64(3.0))

	data := rb.ToSliceFifo()
	expected := []float64{1.0, 2.0, 3.0}
	for i, v := range expected {
		if !data[i].Eq(FromFloat64(v)) {
			t.Errorf("ToSliceFifo()[%d]: got %v, want %v", i, data[i], v)
		}
	}
}

func TestFixedRingBuffer_Mean(t *testing.T) {
	rb := NewRingBuffer(5)

	if !rb.Mean().Eq(Zero) {
		t.Error("Mean of empty buffer should be Zero")
	}

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	for _, v := range values {
		rb.Add(FromFloat64(v))
	}

	assertPointEqual(t, FromFloat64(3.0), rb.Mean(), 0.0001, "Mean calculation")

	rb.Add(FromFloat64(6.0))
	assertPointEqual(t, FromFloat64(4.0), rb.Mean(), 0.0001, "Mean after wraparound")
}

func TestFixedRingBuffer_MinMax(t *testing.T) {
	rb := NewRingBuffer(5)

	t.Run("empty min panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		rb.Min()
	})

	t.Run("empty max panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		rb.Max()
	})

	values := []float64{3.0, 1.0, 4.0, 1.0, 5.0}
	for _, v := range values {
		rb.Add(FromFloat64(v))
	}

	if !rb.Min().Eq(FromFloat64(1.0)) {
		t.Errorf("Min: got %v, want 1.0", rb.Min())
	}

	if !rb.Max().Eq(FromFloat64(5.0)) {
		t.Errorf("Max: got %v, want 5.0", rb.Max())
	}

	rb.Add(FromFloat64(0.0))
	rb.Add(FromFloat64(10.0))

	if !rb.Min().Eq(FromFloat64(0.0)) {
		t.Errorf("Min after wrap: got %v, want 0.0", rb.Min())
	}

	if !rb.Max().Eq(FromFloat64(10.0)) {
		t.Errorf("Max after wrap: got %v, want 10.0", rb.Max())
	}
}

func TestFixedRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	rb.Add(FromFloat64(3.0))

	if rb.Size() != 3 {
		t.Errorf("Size before clear: got %d, want 3", rb.Size())
	}

	rb.Clear()

	if rb.Size() != 0 {
		t.Errorf("Size after clear: got %d, want 0", rb.Size())
	}

	if !rb.IsEmpty() {
		t.Error("Buffer should be empty after clear")
	}

	rb.Add(FromFloat64(4.0))
	if rb.Size() != 1 {
		t.Error("Should be able to add after clear")
	}
}

func TestFixedRingBuffer_ForEachLifo(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	rb.Add(FromFloat64(3.0))

	var collected []float64
	rb.ForEachLifo(func(p Point) {
		pf, _ := p.Float64()
		collected = append(collected, pf)
	})

	expected := []float64{3.0, 2.0, 1.0}
	for i, v := range expected {
		if collected[i] != v {
			t.Errorf("ForEachLifo[%d]: got %v, want %v", i, collected[i], v)
		}
	}
}

func TestFixedRingBuffer_ForEachFifo(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	rb.Add(FromFloat64(3.0))

	var collected []float64
	rb.ForEachFifo(func(p Point) {
		pf, _ := p.Float64()
		collected = append(collected, pf)
	})

	expected := []float64{1.0, 2.0, 3.0}
	for i, v := range expected {
		if collected[i] != v {
			t.Errorf("ForEachFifo[%d]: got %v, want %v", i, collected[i], v)
		}
	}
}

func TestFixedRingBuffer_EdgeCases(t *testing.T) {
	t.Run("single capacity buffer", func(t *testing.T) {
		rb := NewRingBuffer(1)

		rb.Add(FromFloat64(1.0))
		if !rb.IsFull() {
			t.Error("Single capacity buffer should be full after one add")
		}

		rb.Add(FromFloat64(2.0))
		if !rb.Latest().Eq(FromFloat64(2.0)) {
			t.Error("Single capacity buffer should only keep latest")
		}

		if rb.Size() != 1 {
			t.Error("Size should remain 1")
		}
	})

	t.Run("large wraparound", func(t *testing.T) {
		rb := NewRingBuffer(3)

		for i := 1; i <= 10; i++ {
			rb.Add(FromFloat64(float64(i)))
		}

		expected := []float64{10.0, 9.0, 8.0}
		for i, v := range expected {
			if !rb.Get(i).Eq(FromFloat64(v)) {
				t.Errorf("After multiple wraps[%d]: got %v, want %v", i, rb.Get(i), v)
			}
		}
	})
}

func BenchmarkFixedRingBuffer_Add(b *testing.B) {
	rb := NewRingBuffer(100)
	point := FromFloat64(3.14159)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Add(point)
	}
}

func BenchmarkFixedRingBuffer_Get(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Get(i % 100)
	}
}

func BenchmarkFixedRingBuffer_Mean(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Mean()
	}
}

func BenchmarkFixedRingBuffer_StdDev(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.StdDev()
	}
}

func BenchmarkFixedRingBuffer_SampleStdDev(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.SampleStdDev()
	}
}

func BenchmarkFixedRingBuffer_Variance(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Variance()
	}
}

func BenchmarkFixedRingBuffer_SampleVariance(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.SampleVariance()
	}
}

func BenchmarkFixedRingBuffer_ToSliceLifo(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.ToSliceLifo()
	}
}

func BenchmarkFixedRingBuffer_ToSliceFifo(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.ToSliceFifo()
	}
}

func BenchmarkFixedRingBuffer_ForEachLifo(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := Zero
		rb.ForEachLifo(func(p Point) {
			sum = sum.Add(p)
		})
	}
}

func BenchmarkFixedRingBuffer_ForEachFifo(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := Zero
		rb.ForEachFifo(func(p Point) {
			sum = sum.Add(p)
		})
	}
}

func BenchmarkFixedRingBuffer_Min(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Min()
	}
}

func BenchmarkFixedRingBuffer_Max(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Max()
	}
}

func BenchmarkFixedRingBuffer_MinMax(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Min()
		_ = rb.Max()
	}
}

func BenchmarkFixedRingBuffer_Mean_ViaToSliceLifo(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(rb.ToSliceLifo())
	}
}

func BenchmarkFixedRingBuffer_Mean_ViaToSliceFifo(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(rb.ToSliceFifo())
	}
}

func BenchmarkFixedRingBuffer_Mean_Direct(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Mean()
	}
}

func BenchmarkFixedRingBuffer_Add_Large(b *testing.B) {
	rb := NewRingBuffer(10000)
	point := FromFloat64(3.14159)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Add(point)
	}
}

func BenchmarkFixedRingBuffer_StdDev_Large(b *testing.B) {
	rb := NewRingBuffer(10000)
	for i := 0; i < 10000; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.StdDev()
	}
}

func BenchmarkFixedRingBuffer_SampleStdDev_Large(b *testing.B) {
	rb := NewRingBuffer(10000)
	for i := 0; i < 10000; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.SampleStdDev()
	}
}

func BenchmarkFixedRingBuffer_Variance_Large(b *testing.B) {
	rb := NewRingBuffer(10000)
	for i := 0; i < 10000; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Variance()
	}
}

func BenchmarkFixedRingBuffer_SampleVariance_Large(b *testing.B) {
	rb := NewRingBuffer(10000)
	for i := 0; i < 10000; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.SampleVariance()
	}
}
