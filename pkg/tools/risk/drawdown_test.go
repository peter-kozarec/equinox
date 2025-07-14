package risk

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"testing"
)

func TestRisk_withDrawdownCalcSize(t *testing.T) {
	tests := []struct {
		name     string
		baseSize fixed.Point
		drawdown fixed.Point
		want     fixed.Point
	}{
		{
			name:     "no drawdown",
			baseSize: fixed.FromFloat64(1.0),
			drawdown: fixed.FromFloat64(1.0),
			want:     fixed.FromFloat64(1.2),
		},
		{
			name:     "low drawdown",
			baseSize: fixed.FromFloat64(1.0),
			drawdown: fixed.FromFloat64(3.0),
			want:     fixed.FromFloat64(1.0),
		},
		{
			name:     "normal drawdown",
			baseSize: fixed.FromFloat64(1.0),
			drawdown: fixed.FromFloat64(7.0),
			want:     fixed.FromFloat64(0.7),
		},
		{
			name:     "high drawdown",
			baseSize: fixed.FromFloat64(1.0),
			drawdown: fixed.FromFloat64(12.0),
			want:     fixed.FromFloat64(0.5),
		},
		{
			name:     "extreme drawdown",
			baseSize: fixed.FromFloat64(1.0),
			drawdown: fixed.FromFloat64(20.0),
			want:     fixed.FromFloat64(0.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withDrawdownCalcSize(tt.baseSize, tt.drawdown)
			if !got.Eq(tt.want) {
				t.Errorf("withDrawdownCalcSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
