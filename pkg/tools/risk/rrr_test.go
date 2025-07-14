package risk

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"testing"
)

func TestRisk_withRiskRewardRatioCalcSize(t *testing.T) {
	tests := []struct {
		name            string
		baseSize        fixed.Point
		entryPrice      fixed.Point
		stopLossPrice   fixed.Point
		takeProfitPrice fixed.Point
		want            fixed.Point
	}{
		{
			name:            "excellent RRR > 2.5:1 for long",
			baseSize:        fixed.FromFloat64(1.0),
			entryPrice:      fixed.FromFloat64(1.2000),
			stopLossPrice:   fixed.FromFloat64(1.1900),
			takeProfitPrice: fixed.FromFloat64(1.2300),
			want:            fixed.FromFloat64(1.4),
		},
		{
			name:            "excellent RRR > 2.5:1 for short",
			baseSize:        fixed.FromFloat64(1.0),
			entryPrice:      fixed.FromFloat64(1.2000),
			stopLossPrice:   fixed.FromFloat64(1.2100),
			takeProfitPrice: fixed.FromFloat64(1.1700),
			want:            fixed.FromFloat64(1.4),
		},
		{
			name:            "good RRR > 2.0:1",
			baseSize:        fixed.FromFloat64(1.0),
			entryPrice:      fixed.FromFloat64(1.2000),
			stopLossPrice:   fixed.FromFloat64(1.1900),
			takeProfitPrice: fixed.FromFloat64(1.2220),
			want:            fixed.FromFloat64(1.2),
		},
		{
			name:            "acceptable RRR > 1.5:1",
			baseSize:        fixed.FromFloat64(1.0),
			entryPrice:      fixed.FromFloat64(1.2000),
			stopLossPrice:   fixed.FromFloat64(1.1900),
			takeProfitPrice: fixed.FromFloat64(1.2170),
			want:            fixed.FromFloat64(1.0),
		},
		{
			name:            "poor RRR < 1.5:1",
			baseSize:        fixed.FromFloat64(1.0),
			entryPrice:      fixed.FromFloat64(1.2000),
			stopLossPrice:   fixed.FromFloat64(1.1900),
			takeProfitPrice: fixed.FromFloat64(1.2120),
			want:            fixed.FromFloat64(0.8),
		},
		{
			name:            "exact 1.5:1 RRR",
			baseSize:        fixed.FromFloat64(1.0),
			entryPrice:      fixed.FromFloat64(1.2000),
			stopLossPrice:   fixed.FromFloat64(1.1900),
			takeProfitPrice: fixed.FromFloat64(1.2150),
			want:            fixed.FromFloat64(1.0),
		},
		{
			name:            "exact 2.0:1 RRR",
			baseSize:        fixed.FromFloat64(1.0),
			entryPrice:      fixed.FromFloat64(1.2000),
			stopLossPrice:   fixed.FromFloat64(1.1900),
			takeProfitPrice: fixed.FromFloat64(1.2200),
			want:            fixed.FromFloat64(1.2),
		},
		{
			name:            "exact 2.5:1 RRR",
			baseSize:        fixed.FromFloat64(1.0),
			entryPrice:      fixed.FromFloat64(1.2000),
			stopLossPrice:   fixed.FromFloat64(1.1900),
			takeProfitPrice: fixed.FromFloat64(1.2250),
			want:            fixed.FromFloat64(1.4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withRiskRewardRatioCalcSize(tt.baseSize, tt.entryPrice, tt.stopLossPrice, tt.takeProfitPrice)
			if !got.Eq(tt.want) {
				t.Errorf("withRiskRewardRatioCalcSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
