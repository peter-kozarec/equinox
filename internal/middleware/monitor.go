package middleware

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
)

type MonitorFlags uint8

const (
	None         MonitorFlags = 0
	MonitorTicks MonitorFlags = 1 << iota
	MonitorBars
	MonitorEquity
	MonitorBalance
	MonitorPositionsOpened
	MonitorPositionsClosed
	MonitorPositionsPnLUpdated
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
			monitor.logger.Info("event", zap.Any("tick", tick))
		}
		return handler(tick)
	}
}

func (monitor *Monitor) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar *model.Bar) error {
		if monitor.flags&MonitorBars != 0 {
			monitor.logger.Info("event", zap.Any("bar", bar))
		}
		return handler(bar)
	}
}

func (monitor *Monitor) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity *model.Equity) error {
		if monitor.flags&MonitorEquity != 0 {
			monitor.logger.Info("event", zap.Any("equity", equity))
		}
		return handler(equity)
	}
}

func (monitor *Monitor) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance *model.Balance) error {
		if monitor.flags&MonitorBalance != 0 {
			monitor.logger.Info("event", zap.Any("balance", balance))
		}
		return handler(balance)
	}
}

func (monitor *Monitor) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(position *model.Position) error {
		if monitor.flags&MonitorPositionsOpened != 0 {
			monitor.logger.Info("event", zap.Any("position", position))
		}
		return handler(position)
	}
}

func (monitor *Monitor) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position *model.Position) error {
		if monitor.flags&MonitorPositionsClosed != 0 {
			monitor.logger.Info("event", zap.Any("position", position))
		}
		return handler(position)
	}
}

func (monitor *Monitor) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(position *model.Position) error {
		if monitor.flags&MonitorPositionsPnLUpdated != 0 {
			monitor.logger.Info("event", zap.Any("position", position))
		}
		return handler(position)
	}
}
