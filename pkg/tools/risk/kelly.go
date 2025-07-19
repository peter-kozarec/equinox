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
		tradeCountLimit  = 10
		kellyFraction    = 0.25 // Conservative Kelly (25% of full Kelly)
		maxMultiplier    = 3.0
		minMultiplier    = 0.3
		defaultFallback  = 0.5
		maxKellyPosition = 0.25 // Max 25% position size
	)

	return WithKellyMultiplier(func(tradeCount int, winRate, avgWinLoss fixed.Point) fixed.Point {
		if tradeCount < tradeCountLimit {
			return fixed.One
		}

		// Validate inputs
		if avgWinLoss.Lte(fixed.Zero) || winRate.Lte(fixed.Zero) || winRate.Gt(fixed.One) {
			return fixed.FromFloat64(defaultFallback)
		}

		// Kelly formula: f = p - q/b
		// where p = win probability, q = loss probability, b = win/loss ratio
		q := fixed.One.Sub(winRate)
		kellyPercentage := winRate.Sub(q.Div(avgWinLoss))

		// If Kelly suggests no position or negative position
		if kellyPercentage.Lte(fixed.Zero) {
			return fixed.FromFloat64(defaultFallback)
		}

		// Apply conservative fraction (25% of full Kelly)
		conservativeKelly := kellyPercentage.Mul(fixed.FromFloat64(kellyFraction))

		// Cap at maximum position size
		if conservativeKelly.Gt(fixed.FromFloat64(maxKellyPosition)) {
			conservativeKelly = fixed.FromFloat64(maxKellyPosition)
		}

		// Convert to multiplier (assuming base risk of 1%)
		// You might want to make this configurable
		baseRisk := fixed.FromFloat64(0.01)
		multiplier := conservativeKelly.Div(baseRisk)

		// Apply multiplier bounds
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

		// Apply Kelly fraction
		conservativeKelly := kellyPercentage.Mul(fixed.FromFloat64(kellyFraction))

		// Cap at 25% position size
		if conservativeKelly.Gt(fixed.FromFloat64(0.25)) {
			conservativeKelly = fixed.FromFloat64(0.25)
		}

		// Convert to multiplier based on base risk
		baseRisk := fixed.FromFloat64(baseRiskPercentage / 100.0)
		multiplier := conservativeKelly.Div(baseRisk)

		// Apply bounds
		if multiplier.Gt(fixed.FromFloat64(maxMultiplier)) {
			return fixed.FromFloat64(maxMultiplier)
		} else if multiplier.Lt(fixed.FromFloat64(minMultiplier)) {
			return fixed.FromFloat64(minMultiplier)
		}

		return multiplier
	})
}
