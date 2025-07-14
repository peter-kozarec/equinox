package risk

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"testing"
)

func TestRisk_withSignalStrengthCalcSize(t *testing.T) {
	tests := []struct {
		name           string
		baseSize       fixed.Point
		signalStrength uint8
		want           fixed.Point
	}{
		{
			name:           "high signal strength",
			baseSize:       fixed.FromFloat64(1.0),
			signalStrength: 95,
			want:           fixed.FromFloat64(1.0),
		},
		{
			name:           "medium signal strength",
			baseSize:       fixed.FromFloat64(1.0),
			signalStrength: 75,
			want:           fixed.FromFloat64(0.7),
		},
		{
			name:           "low signal strength",
			baseSize:       fixed.FromFloat64(1.0),
			signalStrength: 55,
			want:           fixed.FromFloat64(0.5),
		},
		{
			name:           "no signal strength",
			baseSize:       fixed.FromFloat64(1.0),
			signalStrength: 30,
			want:           fixed.FromFloat64(0.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withSignalStrengthCalcSize(tt.baseSize, tt.signalStrength)
			if !got.Eq(tt.want) {
				t.Errorf("withSignalStrengthCalcSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
