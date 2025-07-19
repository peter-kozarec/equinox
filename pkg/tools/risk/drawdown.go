package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type DrawdownMultiplierHandler func(currentDrawdown fixed.Point) (positionMultiplier fixed.Point)

func WithDrawdownMultiplier(h DrawdownMultiplierHandler) Option {
	return func(m *Manager) {
		if m.drawdownMulHandler != nil {
			panic("drawdown multiplier handler already set")
		}
		m.drawdownMulHandler = h
	}
}

func WithDefaultDrawdownMultiplier() Option {
	var (
		lowDrawdownThreshold     = fixed.FromInt(2, 0)
		normalDrawdownThreshold  = fixed.FromInt(5, 0)
		highDrawdownThreshold    = fixed.FromInt(10, 0)
		extremeDrawdownThreshold = fixed.FromInt(15, 0)

		noDrawdownMultiplier      = fixed.FromFloat64(1.2)
		lowDrawdownMultiplier     = fixed.FromFloat64(1.0)
		normalDrawdownMultiplier  = fixed.FromFloat64(0.7)
		highDrawdownMultiplier    = fixed.FromFloat64(0.5)
		extremeDrawdownMultiplier = fixed.FromFloat64(0.0)
	)

	return WithDrawdownMultiplier(func(currentDrawdown fixed.Point) fixed.Point {
		if currentDrawdown.Lte(lowDrawdownThreshold) {
			return noDrawdownMultiplier
		} else if currentDrawdown.Lte(normalDrawdownThreshold) {
			return lowDrawdownMultiplier
		} else if currentDrawdown.Lte(highDrawdownThreshold) {
			return normalDrawdownMultiplier
		} else if currentDrawdown.Lte(extremeDrawdownThreshold) {
			return highDrawdownMultiplier
		} else {
			return extremeDrawdownMultiplier
		}
	})
}
