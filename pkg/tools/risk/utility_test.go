package risk

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"testing"
)

func TestRisk_clamp(t *testing.T) {
	tests := []struct {
		name string
		p    fixed.Point
		min  fixed.Point
		max  fixed.Point
		want fixed.Point
	}{
		{
			name: "within range",
			p:    fixed.FromFloat64(1.5),
			min:  fixed.FromFloat64(1.0),
			max:  fixed.FromFloat64(2.0),
			want: fixed.FromFloat64(1.5),
		},
		{
			name: "below min",
			p:    fixed.FromFloat64(0.5),
			min:  fixed.FromFloat64(1.0),
			max:  fixed.FromFloat64(2.0),
			want: fixed.FromFloat64(1.0),
		},
		{
			name: "above max",
			p:    fixed.FromFloat64(2.5),
			min:  fixed.FromFloat64(1.0),
			max:  fixed.FromFloat64(2.0),
			want: fixed.FromFloat64(2.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clamp(tt.p, tt.min, tt.max)
			if !got.Eq(tt.want) {
				t.Errorf("clamp() = %v, want %v", got, tt.want)
			}
		})
	}
}
