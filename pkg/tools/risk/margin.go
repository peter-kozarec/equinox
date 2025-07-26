package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

func WithMargin(symbol string, marginPercentage fixed.Point) Option {
	return func(m *Manager) {
		m.margins[symbol] = marginPercentage
	}
}
