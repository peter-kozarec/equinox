package middleware

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
)

type Telemetry struct {
	logger *zap.Logger

	tickEventCounter    int64
	barEventCounter     int64
	balanceEventCounter int64
	equityEventCounter  int64
}

func NewTelemetry(logger *zap.Logger) *Telemetry {
	return &Telemetry{
		logger: logger,
	}
}

func (telemetry *Telemetry) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(tick *model.Tick) error {
		telemetry.tickEventCounter++
		return handler(tick)
	}
}

func (telemetry *Telemetry) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar *model.Bar) error {
		telemetry.barEventCounter++
		return handler(bar)
	}
}

func (telemetry *Telemetry) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance *model.Balance) error {
		telemetry.balanceEventCounter++
		return handler(balance)
	}
}

func (telemetry *Telemetry) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity *model.Equity) error {
		telemetry.equityEventCounter++
		return handler(equity)
	}
}

func (telemetry *Telemetry) PrintStatistics() {
	telemetry.logger.Info("telemetry statistics",
		zap.Int64("tick_events", telemetry.tickEventCounter),
		zap.Int64("bar_events", telemetry.barEventCounter),
		zap.Int64("balance_events", telemetry.balanceEventCounter),
		zap.Int64("equity_events", telemetry.equityEventCounter))
}
