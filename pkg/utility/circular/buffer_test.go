package circular

import "testing"

func TestBuffer_PushGet(t *testing.T) {
	b := NewBuffer[int](5)
	b.Push(0)
	b.Push(1)
	b.Push(2)
	b.Push(3)
	b.Push(4)
	b.Push(5)
	b.Push(6)
	b.Push(7)
	b.Push(8)

	c := NewBuffer[int](8)
	c.Push(0)
	c.Push(1)

	tests := []struct {
		name     string
		result   int
		expected int
	}{
		{"b.Get(0) == 8", b.Get(0), 8},
		{"b.Get(1) == 7", b.Get(1), 7},
		{"b.Get(2) == 6", b.Get(2), 6},
		{"b.Get(3) == 5", b.Get(3), 5},
		{"b.Get(4) == 4", b.Get(4), 4},
		{"b.First() == 8", b.First(), 8},
		{"b.Last() == 4", b.Last(), 4},
		{"c.Get(0) == 1", c.Get(0), 1},
		{"c.Get(1) == 0", c.Get(1), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result != tt.expected {
				t.Errorf("got %d, want %d", tt.result, tt.expected)
			}
		})
	}
}

func TestBuffer_Data(t *testing.T) {
	b := NewBuffer[int](5)
	b.Push(0)
	b.Push(1)
	b.Push(2)
	b.Push(3)
	b.Push(4)
	b.Push(5)
	b.Push(6)
	b.Push(7)
	b.Push(8)

	tests := []struct {
		name     string
		result   []int
		expected []int
	}{
		{"b.Data() == [4,5,6,7,8]", b.Data(), []int{4, 5, 6, 7, 8}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !compareSlices(tt.result, tt.expected) {
				t.Errorf("got %d, want %d", tt.result, tt.expected)
			}
		})
	}
}

func compareSlices(slice1, slice2 []int) bool {
	// Check if the lengths are equal
	if len(slice1) != len(slice2) {
		return false
	}

	// Compare each element
	for i := range slice1 {
		if slice1[i] != slice2[i] {
			return false
		}
	}

	return true
}
