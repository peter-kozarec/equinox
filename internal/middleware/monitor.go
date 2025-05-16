package middleware

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
)

type Monitor struct {
	logger *zap.Logger
}

func NewMonitor(logger *zap.Logger) *Monitor {
	return &Monitor{
		logger: logger,
	}
}

func (monitor *Monitor) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(tick *model.Tick) error {
		monitor.logger.Debug("monitor", zap.Any("tick", tick))
		return handler(tick)
	}
}

func (monitor *Monitor) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar *model.Bar) error {
		monitor.logger.Debug("monitor", zap.Any("bar", bar))
		return handler(bar)
	}
}

func (monitor *Monitor) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity *model.Equity) error {
		monitor.logger.Debug("monitor", zap.Any("equity", equity))
		return handler(equity)
	}
}

func (monitor *Monitor) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance *model.Balance) error {
		monitor.logger.Debug("monitor", zap.Any("balance", balance))
		return handler(balance)
	}
}
