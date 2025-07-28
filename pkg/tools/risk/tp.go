package risk

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type TakeProfitHandler func(signal common.Signal) fixed.Point

func WithTakeProfit(h TakeProfitHandler) Option {
	return func(m *Manager) {
		m.takeProfitHandler = h
	}
}

func WithFixedTakeProfit() Option {
	return WithTakeProfit(func(signal common.Signal) fixed.Point {
		return signal.Target
	})
}
