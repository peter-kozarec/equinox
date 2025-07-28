package risk

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/tools/indicators"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type StopLossHandler func(signal common.Signal, spread, atr fixed.Point) fixed.Point

func WithStopLoss(atrWindow int, h StopLossHandler) Option {
	return func(m *Manager) {
		if m.slAtr != nil {
			panic("slAtr already set")
		}
		m.slAtr = indicators.NewAtr(atrWindow)

		if m.stopLossHandler != nil {
			panic("stop loss handler already set")
		}
		m.stopLossHandler = h
	}
}

func WithAtrStopLoss(atrWindow int, atrMul fixed.Point) Option {
	return WithStopLoss(atrWindow, func(signal common.Signal, spread, atr fixed.Point) fixed.Point {
		slDistance := signal.Entry.Sub(atr.Mul(atrMul)).Abs()
		if signal.Entry.Gt(signal.Target) {
			return signal.Entry.Add(slDistance)
		}
		return signal.Entry.Sub(slDistance)
	})
}
