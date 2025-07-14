package risk

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"testing"
)

func TestRisk_withKellyCriterionCalcSize(t *testing.T) {
	tests := []struct {
		name               string
		baseSize           fixed.Point
		winRate            fixed.Point
		avgWinLoss         fixed.Point
		baseRiskPercentage fixed.Point
		want               fixed.Point
	}{
		{
			name:               "positive edge - 60% win rate, 1.5:1 ratio",
			baseSize:           fixed.FromFloat64(1.0),
			winRate:            fixed.FromFloat64(0.6),
			avgWinLoss:         fixed.FromFloat64(1.5),
			baseRiskPercentage: fixed.FromFloat64(1.0),
			want:               fixed.FromFloat64(3.0),
		},
		{
			name:               "moderate edge - 55% win rate, 1.2:1 ratio",
			baseSize:           fixed.FromFloat64(1.0),
			winRate:            fixed.FromFloat64(0.55),
			avgWinLoss:         fixed.FromFloat64(1.2),
			baseRiskPercentage: fixed.FromFloat64(1.0),
			want:               fixed.FromFloat64(3),
		},
		{
			name:               "small edge - 52% win rate, 1.1:1 ratio",
			baseSize:           fixed.FromFloat64(1.0),
			winRate:            fixed.FromFloat64(0.52),
			avgWinLoss:         fixed.FromFloat64(1.1),
			baseRiskPercentage: fixed.FromFloat64(1.0),
			want:               fixed.FromInt64(209090909090909091, 17),
		},
		{
			name:               "negative edge - 40% win rate, 1:1 ratio",
			baseSize:           fixed.FromFloat64(1.0),
			winRate:            fixed.FromFloat64(0.4),
			avgWinLoss:         fixed.FromFloat64(1.0),
			baseRiskPercentage: fixed.FromFloat64(1.0),
			want:               fixed.FromFloat64(0.5),
		},
		{
			name:               "high base risk - 2% base",
			baseSize:           fixed.FromFloat64(1.0),
			winRate:            fixed.FromFloat64(0.6),
			avgWinLoss:         fixed.FromFloat64(1.5),
			baseRiskPercentage: fixed.FromFloat64(2.0),
			want:               fixed.FromFloat64(3.0),
		},
		{
			name:               "invalid win rate",
			baseSize:           fixed.FromFloat64(1.0),
			winRate:            fixed.FromFloat64(1.5),
			avgWinLoss:         fixed.FromFloat64(2.0),
			baseRiskPercentage: fixed.FromFloat64(1.0),
			want:               fixed.FromFloat64(0.5),
		},
		{
			name:               "zero avg win/loss",
			baseSize:           fixed.FromFloat64(1.0),
			winRate:            fixed.FromFloat64(0.6),
			avgWinLoss:         fixed.Zero,
			baseRiskPercentage: fixed.FromFloat64(1.0),
			want:               fixed.FromFloat64(0.5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withKellyCriterionCalcSize(tt.baseSize, tt.winRate, tt.avgWinLoss, tt.baseRiskPercentage)
			if !got.Eq(tt.want) {
				// Debug output
				q := fixed.One.Sub(tt.winRate)
				kelly := tt.winRate.Sub(q.Div(tt.avgWinLoss))
				kellySafety := kelly.Mul(fixed.FromFloat64(0.25))
				multiplier := kellySafety.Div(tt.baseRiskPercentage.DivInt(100))

				t.Errorf("withKellyCriterionCalcSize() = %v, want %v\n"+
					"Kelly: %v, Kelly*0.25: %v, Multiplier: %v",
					got, tt.want, kelly, kellySafety, multiplier)
			}
		})
	}
}
