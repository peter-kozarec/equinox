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

func TestFixedRing_NewRingBuffer(t *testing.T) {
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

func TestFixedRing_Add(t *testing.T) {
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

func TestFixedRing_Get(t *testing.T) {
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

func TestFixedRing_IsEmpty(t *testing.T) {
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

func TestFixedRing_IsFull(t *testing.T) {
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

func TestFixedRing_Latest(t *testing.T) {
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

func TestFixedRing_Oldest(t *testing.T) {
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

func TestFixedRing_Data(t *testing.T) {
	rb := NewRingBuffer(3)

	if data := rb.Data(); data != nil {
		t.Error("Data() should return nil for empty buffer")
	}

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	data := rb.Data()
	expected := []float64{2.0, 1.0}
	for i, v := range expected {
		if !data[i].Eq(FromFloat64(v)) {
			t.Errorf("Data()[%d]: got %v, want %v", i, data[i], v)
		}
	}

	rb.Add(FromFloat64(3.0))
	rb.Add(FromFloat64(4.0))
	data = rb.Data()
	expected = []float64{4.0, 3.0, 2.0}
	for i, v := range expected {
		if !data[i].Eq(FromFloat64(v)) {
			t.Errorf("Data()[%d]: got %v, want %v", i, data[i], v)
		}
	}
}

func TestFixedRing_DataReversed(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	rb.Add(FromFloat64(3.0))

	data := rb.DataReversed()
	expected := []float64{1.0, 2.0, 3.0}
	for i, v := range expected {
		if !data[i].Eq(FromFloat64(v)) {
			t.Errorf("DataReversed()[%d]: got %v, want %v", i, data[i], v)
		}
	}
}

func TestFixedRing_Mean(t *testing.T) {
	rb := NewRingBuffer(5)

	if !rb.Mean().Eq(Zero) {
		t.Error("Mean of empty buffer should be Zero")
	}

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	for _, v := range values {
		rb.Add(FromFloat64(v))
	}

	assertPointEqual(t, FromFloat64(3.0), rb.Mean(), 0.0001, "Mean calculation")

	// After wraparound
	rb.Add(FromFloat64(6.0))
	assertPointEqual(t, FromFloat64(4.0), rb.Mean(), 0.0001, "Mean after wraparound")
}

func TestFixedRing_MinMax(t *testing.T) {
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

func TestFixedRing_Clear(t *testing.T) {
	rb := NewRingBuffer(3)

	// Fill buffer
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

func TestFixedRing_ForEach(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	rb.Add(FromFloat64(3.0))

	var collected []float64
	rb.ForEach(func(p Point) {
		pf, _ := p.Float64()
		collected = append(collected, pf)
	})

	expected := []float64{3.0, 2.0, 1.0}
	for i, v := range expected {
		if collected[i] != v {
			t.Errorf("ForEach[%d]: got %v, want %v", i, collected[i], v)
		}
	}
}

func TestFixedRing_ForEachReversed(t *testing.T) {
	rb := NewRingBuffer(3)

	rb.Add(FromFloat64(1.0))
	rb.Add(FromFloat64(2.0))
	rb.Add(FromFloat64(3.0))

	var collected []float64
	rb.ForEachReversed(func(p Point) {
		pf, _ := p.Float64()
		collected = append(collected, pf)
	})

	expected := []float64{1.0, 2.0, 3.0}
	for i, v := range expected {
		if collected[i] != v {
			t.Errorf("ForEachReversed[%d]: got %v, want %v", i, collected[i], v)
		}
	}
}

func TestFixedRing_EdgeCases(t *testing.T) {
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

func BenchmarkFixedRing_Add(b *testing.B) {
	rb := NewRingBuffer(100)
	point := FromFloat64(3.14159)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Add(point)
	}
}

func BenchmarkFixedRing_Get(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Get(i % 100)
	}
}

func BenchmarkFixedRing_Mean(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Mean()
	}
}

func BenchmarkFixedRing_StdDev(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.StdDev()
	}
}

func BenchmarkFixedRing_Data(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Data()
	}
}

func BenchmarkFixedRing_ForEach(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := Zero
		rb.ForEach(func(p Point) {
			sum = sum.Add(p)
		})
	}
}

func BenchmarkFixedRing_MinMax(b *testing.B) {
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

func BenchmarkFixedRing_Mean_ViaData(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mean(rb.Data())
	}
}

func BenchmarkFixedRing_Mean_Direct(b *testing.B) {
	rb := NewRingBuffer(100)
	for i := 0; i < 100; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Mean()
	}
}

func BenchmarkFixedRing_Add_Large(b *testing.B) {
	rb := NewRingBuffer(10000)
	point := FromFloat64(3.14159)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Add(point)
	}
}

func BenchmarkFixedRing_StdDev_Large(b *testing.B) {
	rb := NewRingBuffer(10000)
	for i := 0; i < 10000; i++ {
		rb.Add(FromFloat64(float64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.StdDev()
	}
}
