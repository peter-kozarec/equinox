package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

func WithTrailingStopLoss(trailingDistancePercentage, trailingMovePercentage fixed.Point) Option {
	return func(m *Manager) {
		m.trailingDistance = trailingDistancePercentage.DivInt(100)
		m.trailingMove = trailingMovePercentage.DivInt(100)
	}
}
