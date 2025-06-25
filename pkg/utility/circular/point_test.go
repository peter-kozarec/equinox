package circular

import (
	"testing"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	zero  = fixed.New(0, 0)
	one   = fixed.New(1, 0)
	two   = fixed.New(2, 0)
	three = fixed.New(3, 0)
	four  = fixed.New(4, 0)
	ten   = fixed.New(10, 0)
)

func TestPoint(t *testing.T) {
	p := CreatePointBuffer(5)
	p.PushUpdate(three)
	p.PushUpdate(one)
	p.PushUpdate(two)
	p.PushUpdate(zero)
	p.PushUpdate(one)
	p.PushUpdate(two)
	p.PushUpdate(three)
	p.PushUpdate(four)

	tests := []struct {
		name     string
		result   fixed.Point
		expected fixed.Point
	}{
		{"p.Mean() == 2.0", p.Mean(), two},
		{"p.Sum() == 10.0", p.Sum(), ten},
		{"p.StdDev() == 1.4142", p.StdDev(), two.Sqrt()},
		{"p.Variance() == 2.0", p.Variance(), two},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result != tt.expected {
				t.Errorf("got %d, want %d", tt.result, tt.expected)
			}
		})
	}
}
