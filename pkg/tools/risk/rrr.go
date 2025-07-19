package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type RRRMultiplierHandler func(rrr fixed.Point) fixed.Point

func WithRRRMultiplier(h RRRMultiplierHandler) Option {
	return func(m *Manager) {
		if m.rrrMulHandler != nil {
			panic("RRR multiplier handler already set")
		}
		m.rrrMulHandler = h
	}
}

func WithDefaultRRRMultiplier() Option {
	return WithRRRMultiplier(func(rrr fixed.Point) fixed.Point {
		if rrr.Gte(fixed.FromFloat64(2.5)) {
			return fixed.FromFloat64(1.4)
		} else if rrr.Gte(fixed.FromFloat64(2.0)) {
			return fixed.FromFloat64(1.2)
		} else if rrr.Gte(fixed.FromFloat64(1.5)) {
			return fixed.FromFloat64(1.0)
		}
		return fixed.FromFloat64(0.8)
	})
}
