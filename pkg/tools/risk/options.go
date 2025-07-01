package risk

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type ManagerOption func(*Manager)

func WithSydneySessionMultiplier(mul fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.fxSession.sydneyMultiplier = mul
	}
}

func WithTokyoSessionMultiplier(mul fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.fxSession.tokyoMultiplier = mul
	}
}

func WithLondonSessionMultiplier(mul fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.fxSession.londonMultiplier = mul
	}
}

func WithNewYorkSessionMultiplier(mul fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.fxSession.newYorkMultiplier = mul
	}
}

func WithSydneyTokyoSessionMultiplier(mul fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.fxSession.sydneyTokyoMultiplier = mul
	}
}

func WithTokyoLondonSessionMultiplier(mul fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.fxSession.tokyoLondonMultiplier = mul
	}
}

func WithLondonNewYorkSessionMultiplier(mul fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.fxSession.londonNewYorkMultiplier = mul
	}
}

type DrawdownMultiplierFunc func(currentDrawdown fixed.Point) fixed.Point

func WithDefaultDrawdownMultiplier() ManagerOption {
	return WithDrawdownMultiplier(func(currentDrawdown fixed.Point) fixed.Point {
		switch {
		case currentDrawdown.Lt(fixed.New(5, 0)):
			return fixed.One
		case currentDrawdown.Lt(fixed.New(10, 0)):
			return fixed.FromFloat(0.8)
		case currentDrawdown.Lt(fixed.New(15, 0)):
			return fixed.FromFloat(0.6)
		case currentDrawdown.Lt(fixed.New(20, 0)):
			return fixed.FromFloat(0.4)
		case currentDrawdown.Lt(fixed.New(25, 0)):
			return fixed.FromFloat(0.2)
		default:
			return fixed.Zero
		}
	})
}

func WithDrawdownMultiplier(ddMulFunc DrawdownMultiplierFunc) ManagerOption {
	return func(m *Manager) {
		m.account.drawdownFunc = ddMulFunc
	}
}

func WithVolatilityMultiplier(lowVolMul, normVolMul, highVolMul fixed.Point) ManagerOption {

	return func(m *Manager) {
		m.volatility.lowVolMultiplier = lowVolMul
		m.volatility.normVolMultiplier = normVolMul
		m.volatility.highVolMultiplier = highVolMul
	}
}

func WithLeverage(leverage fixed.Point) ManagerOption {
	return func(m *Manager) {
		m.account.leverage = leverage
	}
}
