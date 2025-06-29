package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type ManagerOption func(*Manager)

func WithRiskPercentage(riskPercentage fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.riskPercentage = riskPercentage
	}
}

func WithVolatilityFactor(factor bool) ManagerOption {
	return func(m *Manager) {
		m.volatilityFactor = factor
	}
}

func WithDrawdownFactor(factor bool) ManagerOption {
	return func(m *Manager) {
		m.drawdownFactor = factor
	}
}

func WithSignalConfidenceFactor(factor bool) ManagerOption {
	return func(m *Manager) {
		m.signalConfidenceFactor = factor
	}
}
