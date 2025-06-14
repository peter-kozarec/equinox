package middleware

import (
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/model"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type MonitorFlags uint16

const (
	MonitorTicks MonitorFlags = 1 << iota
	MonitorBars
	MonitorEquity
	MonitorBalance
	MonitorPositionsOpened
	MonitorPositionsClosed
	MonitorPositionsPnLUpdated
	MonitorOrders
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

func (monitor *Monitor) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(tick *model.Tick) error {
		if monitor.flags&MonitorTicks != 0 {
			monitor.logger.Info("event", zap.Object("tick", tick))
		}
		return handler(tick)
	}
}

func (monitor *Monitor) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar *model.Bar) error {
		if monitor.flags&MonitorBars != 0 {
			monitor.logger.Info("bar event", zap.Object("bar", bar))
		}
		return handler(bar)
	}
}

func (monitor *Monitor) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity *fixed.Point) error {
		if monitor.flags&MonitorEquity != 0 {
			monitor.logger.Info("equity event", zap.Object("equity", equity))
		}
		return handler(equity)
	}
}

func (monitor *Monitor) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance *fixed.Point) error {
		if monitor.flags&MonitorBalance != 0 {
			monitor.logger.Info("balance event", zap.Object("balance", balance))
		}
		return handler(balance)
	}
}

func (monitor *Monitor) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(position *model.Position) error {
		if monitor.flags&MonitorPositionsOpened != 0 {
			monitor.logger.Info("position opened event", zap.Object("position", position))
		}
		return handler(position)
	}
}

func (monitor *Monitor) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position *model.Position) error {
		if monitor.flags&MonitorPositionsClosed != 0 {
			monitor.logger.Info("position closed event", zap.Object("position", position))
		}
		return handler(position)
	}
}

func (monitor *Monitor) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(position *model.Position) error {
		if monitor.flags&MonitorPositionsPnLUpdated != 0 {
			monitor.logger.Info("position pnl updated event", zap.Object("position", position))
		}
		return handler(position)
	}
}

func (monitor *Monitor) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(order *model.Order) error {
		if monitor.flags&MonitorOrders != 0 {
			monitor.logger.Info("order event", zap.Object("order", order))
		}
		return handler(order)
	}
}
