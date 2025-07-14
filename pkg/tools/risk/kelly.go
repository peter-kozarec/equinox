package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

func withKellyCriterionCalcSize(baseSize, winRate, avgWinLoss, baseRiskPercentage fixed.Point) fixed.Point {
	// Kelly formula: f = p - q/b
	// where p = win probability, q = loss probability, b = win/loss ratio

	if avgWinLoss.Lte(fixed.Zero) || winRate.Lte(fixed.Zero) || winRate.Gt(fixed.One) {
		return baseSize.Mul(fixed.FromFloat64(0.5))
	}

	q := fixed.One.Sub(winRate)
	kellyPercentage := winRate.Sub(q.Div(avgWinLoss))

	if kellyPercentage.Lte(fixed.Zero) {
		return baseSize.Mul(fixed.FromFloat64(0.5))
	}

	kellyWithSafety := kellyPercentage.Mul(fixed.FromFloat64(0.25))

	if kellyWithSafety.Gt(fixed.FromFloat64(0.25)) {
		kellyWithSafety = fixed.FromFloat64(0.25)
	}

	multiplier := kellyWithSafety.Div(baseRiskPercentage.DivInt(100))

	if multiplier.Gt(fixed.FromFloat64(3.0)) {
		multiplier = fixed.FromFloat64(3.0)
	} else if multiplier.Lt(fixed.FromFloat64(0.3)) {
		multiplier = fixed.FromFloat64(0.3)
	}

	return baseSize.Mul(multiplier)
}
