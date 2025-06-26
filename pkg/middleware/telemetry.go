package middleware

import (
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
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

func (t *Telemetry) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(tick common.Tick) {
		t.tickEventCounter++
		handler(tick)
	}
}

func (t *Telemetry) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar common.Bar) {
		t.barEventCounter++
		handler(bar)
	}
}

func (t *Telemetry) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance fixed.Point) {
		t.balanceEventCounter++
		handler(balance)
	}
}

func (t *Telemetry) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity fixed.Point) {
		t.equityEventCounter++
		handler(equity)
	}
}

func (t *Telemetry) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(position common.Position) {
		t.positionOpenedEventCounter++
		handler(position)
	}
}

func (t *Telemetry) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position common.Position) {
		t.positionClosedEventCounter++
		handler(position)
	}
}

func (t *Telemetry) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(position common.Position) {
		t.positionPnLUpdatedEventCounter++
		handler(position)
	}
}

func (t *Telemetry) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(order common.Order) {
		t.orderEventCounter++
		handler(order)
	}
}

func (t *Telemetry) PrintStatistics() {
	t.logger.Info("event statistics",
		zap.Uint64("tick_events", t.tickEventCounter),
		zap.Uint64("bar_events", t.barEventCounter),
		zap.Uint64("balance_events", t.balanceEventCounter),
		zap.Uint64("equity_events", t.equityEventCounter),
		zap.Uint64("position_opened_events", t.positionOpenedEventCounter),
		zap.Uint64("position_closed_events", t.positionClosedEventCounter),
		zap.Uint64("position_pnl_updated_events", t.positionPnLUpdatedEventCounter),
		zap.Uint64("order_events", t.orderEventCounter))
}
