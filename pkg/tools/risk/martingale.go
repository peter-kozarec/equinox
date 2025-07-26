package risk

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type MartingaleMultiplierHandler func(positions []common.Position) fixed.Point

func WithMartingaleMultiplier(h MartingaleMultiplierHandler) Option {
	return func(m *Manager) {
		m.martingaleMulHandler = h
	}
}

func WithDefaultMartingaleMultiplier(martingaleMultiplier fixed.Point) Option {
	return WithMartingaleMultiplier(func(positions []common.Position) fixed.Point {
		multiplier := fixed.One
		for i := len(positions) - 1; i >= 0; i-- {
			if positions[i].GrossProfit.Gte(fixed.Zero) {
				break
			}
			multiplier = multiplier.Mul(martingaleMultiplier)
		}
		return multiplier
	})
}
