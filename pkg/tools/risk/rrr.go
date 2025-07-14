package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

func withRiskRewardRatioCalcSize(baseSize, entry, sl, tp fixed.Point) fixed.Point {
	risk := entry.Sub(sl).Abs()
	reward := tp.Sub(entry).Abs()
	ratio := reward.Div(risk)

	if ratio.Gte(fixed.FromFloat64(2.5)) {
		return baseSize.Mul(fixed.FromFloat64(1.4))
	} else if ratio.Gte(fixed.FromFloat64(2.0)) {
		return baseSize.Mul(fixed.FromFloat64(1.2))
	} else if ratio.Gte(fixed.FromFloat64(1.5)) {
		return baseSize.Mul(fixed.FromFloat64(1.0))
	}

	return baseSize.Mul(fixed.FromFloat64(0.8))
}
