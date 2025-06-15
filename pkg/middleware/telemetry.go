package middleware

import (
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/model"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type Telemetry struct {
	logger *zap.Logger

	tickEventCounter               uint64
	barEventCounter                uint64
	balanceEventCounter            uint64
	equityEventCounter             uint64
	positionOpenedEventCounter     uint64
	positionClosedEventCounter     uint64
	positionPnLUpdatedEventCounter uint64
	orderEventCounter              uint64
}

func NewTelemetry(logger *zap.Logger) *Telemetry {
	return &Telemetry{
		logger: logger,
	}
}

func (telemetry *Telemetry) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(tick model.Tick) {
		telemetry.tickEventCounter++
		handler(tick)
	}
}

func (telemetry *Telemetry) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar model.Bar) {
		telemetry.barEventCounter++
		handler(bar)
	}
}

func (telemetry *Telemetry) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance fixed.Point) {
		telemetry.balanceEventCounter++
		handler(balance)
	}
}

func (telemetry *Telemetry) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity fixed.Point) {
		telemetry.equityEventCounter++
		handler(equity)
	}
}

func (telemetry *Telemetry) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(position model.Position) {
		telemetry.positionOpenedEventCounter++
		handler(position)
	}
}

func (telemetry *Telemetry) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position model.Position) {
		telemetry.positionClosedEventCounter++
		handler(position)
	}
}

func (telemetry *Telemetry) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(position model.Position) {
		telemetry.positionPnLUpdatedEventCounter++
		handler(position)
	}
}

func (telemetry *Telemetry) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(order model.Order) {
		telemetry.orderEventCounter++
		handler(order)
	}
}

func (telemetry *Telemetry) PrintStatistics() {
	telemetry.logger.Info("event statistics",
		zap.Uint64("tick_events", telemetry.tickEventCounter),
		zap.Uint64("bar_events", telemetry.barEventCounter),
		zap.Uint64("balance_events", telemetry.balanceEventCounter),
		zap.Uint64("equity_events", telemetry.equityEventCounter),
		zap.Uint64("position_opened_events", telemetry.positionOpenedEventCounter),
		zap.Uint64("position_closed_events", telemetry.positionClosedEventCounter),
		zap.Uint64("position_pnl_updated_events", telemetry.positionPnLUpdatedEventCounter),
		zap.Uint64("order_events", telemetry.orderEventCounter))
}
