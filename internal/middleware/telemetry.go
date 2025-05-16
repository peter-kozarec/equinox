package middleware

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
)

type Telemetry struct {
	logger *zap.Logger

	tickEventCounter               int64
	barEventCounter                int64
	balanceEventCounter            int64
	equityEventCounter             int64
	positionOpenedEventCounter     int64
	positionClosedEventCounter     int64
	positionPnLUpdatedEventCounter int64
	orderEventCounter              int64
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

func (telemetry *Telemetry) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(position *model.Position) error {
		telemetry.positionOpenedEventCounter++
		return handler(position)
	}
}

func (telemetry *Telemetry) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position *model.Position) error {
		telemetry.positionClosedEventCounter++
		return handler(position)
	}
}

func (telemetry *Telemetry) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(position *model.Position) error {
		telemetry.positionPnLUpdatedEventCounter++
		return handler(position)
	}
}

func (telemetry *Telemetry) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(order *model.Order) error {
		telemetry.orderEventCounter++
		return handler(order)
	}
}

func (telemetry *Telemetry) PrintStatistics() {
	telemetry.logger.Info("telemetry statistics",
		zap.Int64("tick_events", telemetry.tickEventCounter),
		zap.Int64("bar_events", telemetry.barEventCounter),
		zap.Int64("balance_events", telemetry.balanceEventCounter),
		zap.Int64("equity_events", telemetry.equityEventCounter),
		zap.Int64("position_opened_events", telemetry.positionOpenedEventCounter),
		zap.Int64("position_closed_events", telemetry.positionClosedEventCounter),
		zap.Int64("position_pnl_updated_events", telemetry.positionPnLUpdatedEventCounter),
		zap.Int64("order_events", telemetry.orderEventCounter))
}
