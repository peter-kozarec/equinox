package risk

import "github.com/peter-kozarec/equinox/pkg/tools/risk/adjustment"

func WithAdjustment(adj adjustment.DynamicAdjustment) Option {
	return func(m *Manager) {
		m.adj = adj
	}
}
