package middleware

import (
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type MonitorFlags uint16

//goland:noinspection GoUnusedConst
const (
	MonitorNone MonitorFlags = 1 << iota
	MonitorTicks
	MonitorBars
	MonitorEquity
	MonitorBalance
	MonitorPositionsOpened
	MonitorPositionsClosed
	MonitorPositionsPnLUpdated
	MonitorOrders
	MonitorSignals
)

type Monitor struct {
	logger *zap.Logger
	flags  MonitorFlags
}

func NewMonitor(logger *zap.Logger, flags MonitorFlags) *Monitor {
	return &Monitor{
		logger: logger,
		flags:  flags,
	}
}

func (m *Monitor) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(tick common.Tick) {
		if m.flags&MonitorTicks != 0 {
			m.logger.Info("event", zap.Any("tick", tick.Fields()))
		}
		handler(tick)
	}
}

func (m *Monitor) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar common.Bar) {
		if m.flags&MonitorBars != 0 {
			m.logger.Info("event", zap.Any("bar", bar.Fields()))
		}
		handler(bar)
	}
}

func (m *Monitor) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity fixed.Point) {
		if m.flags&MonitorEquity != 0 {
			m.logger.Info("event", zap.String("equity", equity.String()))
		}
		handler(equity)
	}
}

func (m *Monitor) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance fixed.Point) {
		if m.flags&MonitorBalance != 0 {
			m.logger.Info("event", zap.String("balance", balance.String()))
		}
		handler(balance)
	}
}

func (m *Monitor) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(position common.Position) {
		if m.flags&MonitorPositionsOpened != 0 {
			m.logger.Info("event", zap.Any("position_open", position.Fields()))
		}
		handler(position)
	}
}

func (m *Monitor) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position common.Position) {
		if m.flags&MonitorPositionsClosed != 0 {
			m.logger.Info("event", zap.Any("position_closed", position.Fields()))
		}
		handler(position)
	}
}

func (m *Monitor) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(position common.Position) {
		if m.flags&MonitorPositionsPnLUpdated != 0 {
			m.logger.Info("event", zap.Any("position", position.Fields()))
		}
		handler(position)
	}
}

func (m *Monitor) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(order common.Order) {
		if m.flags&MonitorOrders != 0 {
			m.logger.Info("event", zap.Any("order", order.Fields()))
		}
		handler(order)
	}
}

func (m *Monitor) WithSignal(handler bus.SignalEventHandler) bus.SignalEventHandler {
	return func(signal common.Signal) {
		if m.flags&MonitorSignals != 0 {
			m.logger.Info("event", zap.Any("signal", signal.Fields()))
		}
		handler(signal)
	}
}
