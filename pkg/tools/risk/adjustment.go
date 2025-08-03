package risk

import "github.com/peter-kozarec/equinox/pkg/common"

type AdjustmentHandler interface {
	AdjustPosition(common.Position) (common.Order, bool)
}

func WithAdjustment(handler AdjustmentHandler) Option {
	return func(m *Manager) {
		m.adjustmentHandler = handler
	}
}
