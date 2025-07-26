package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type KellyMultiplierHandler func(tradeCount int, winRate, avgWinLoss fixed.Point) fixed.Point

func WithKellyMultiplier(h KellyMultiplierHandler) Option {
	return func(m *Manager) {
		if m.kellyMulHandler != nil {
			panic("Kelly multiplier handler already set")
		}
		m.kellyMulHandler = h
	}
}

func WithDefaultKellyMultiplier() Option {
	const (
		tradeCountLimit      = 10
		kellyFraction        = 0.25
		maxMultiplier        = 3.0
		minMultiplier        = 0.3
		defaultFallback      = 1.0
		noPositionMultiplier = 0.0
		maxKellyPosition     = 0.25
	)

	return WithKellyMultiplier(func(tradeCount int, winRate, avgWinLoss fixed.Point) fixed.Point {
		if tradeCount < tradeCountLimit {
			return fixed.One
		}

		// Validate inputs
		if avgWinLoss.Lte(fixed.Zero) || winRate.Lte(fixed.Zero) || winRate.Gt(fixed.One) {
			return fixed.FromFloat64(defaultFallback)
		}

		q := fixed.One.Sub(winRate)
		kellyPercentage := winRate.Sub(q.Div(avgWinLoss))

		if kellyPercentage.Lte(fixed.Zero) {
			if winRate.Lt(fixed.FromFloat64(0.4)) {
				return fixed.FromFloat64(noPositionMultiplier)
			}
			return fixed.FromFloat64(minMultiplier)
		}

		conservativeKelly := kellyPercentage.Mul(fixed.FromFloat64(kellyFraction))

		if conservativeKelly.Gt(fixed.FromFloat64(maxKellyPosition)) {
			conservativeKelly = fixed.FromFloat64(maxKellyPosition)
		}

		baseRisk := fixed.FromFloat64(0.01)
		multiplier := conservativeKelly.Div(baseRisk)

		if multiplier.Gt(fixed.FromFloat64(maxMultiplier)) {
			return fixed.FromFloat64(maxMultiplier)
		} else if multiplier.Lt(fixed.FromFloat64(minMultiplier)) {
			return fixed.FromFloat64(minMultiplier)
		}

		return multiplier
	})
}

func WithConfigurableKellyMultiplier(tradeCountLimit int, kellyFraction, maxMultiplier, minMultiplier, baseRiskPercentage float64) Option {
	return WithKellyMultiplier(func(tradeCount int, winRate, avgWinLoss fixed.Point) fixed.Point {
		if tradeCount < tradeCountLimit {
			return fixed.One
		}

		if avgWinLoss.Lte(fixed.Zero) || winRate.Lte(fixed.Zero) || winRate.Gt(fixed.One) {
			return fixed.FromFloat64(minMultiplier)
		}

		q := fixed.One.Sub(winRate)
		kellyPercentage := winRate.Sub(q.Div(avgWinLoss))

		if kellyPercentage.Lte(fixed.Zero) {
			return fixed.FromFloat64(minMultiplier)
		}

		conservativeKelly := kellyPercentage.Mul(fixed.FromFloat64(kellyFraction))

		if conservativeKelly.Gt(fixed.FromFloat64(0.25)) {
			conservativeKelly = fixed.FromFloat64(0.25)
		}

		baseRisk := fixed.FromFloat64(baseRiskPercentage / 100.0)
		multiplier := conservativeKelly.Div(baseRisk)

		if multiplier.Gt(fixed.FromFloat64(maxMultiplier)) {
			return fixed.FromFloat64(maxMultiplier)
		} else if multiplier.Lt(fixed.FromFloat64(minMultiplier)) {
			return fixed.FromFloat64(minMultiplier)
		}

		return multiplier
	})
}
