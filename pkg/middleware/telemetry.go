package middleware

import (
	"log/slog"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Telemetry struct {
	tickEventCounter               int64
	barEventCounter                int64
	balanceEventCounter            int64
	equityEventCounter             int64
	positionOpenedEventCounter     int64
	positionClosedEventCounter     int64
	positionPnLUpdatedEventCounter int64
	orderEventCounter              int64
	signalEventCounter             int64
}

func NewTelemetry() *Telemetry {
	return &Telemetry{}
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

func (t *Telemetry) WithSignal(handler bus.SignalEventHandler) bus.SignalEventHandler {
	return func(signal common.Signal) {
		t.signalEventCounter++
		handler(signal)
	}
}

func (t *Telemetry) PrintStatistics() {
	slog.Info("event statistics",
		"tick_events", t.tickEventCounter,
		"bar_events", t.barEventCounter,
		"balance_events", t.balanceEventCounter,
		"equity_events", t.equityEventCounter,
		"position_opened_events", t.positionOpenedEventCounter,
		"position_closed_events", t.positionClosedEventCounter,
		"position_pnl_updated_events", t.positionPnLUpdatedEventCounter,
		"order_events", t.orderEventCounter,
		"signal_events", t.signalEventCounter)
}
