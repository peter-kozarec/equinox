package middleware

import (
	"context"
	"log/slog"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
)

type MonitorFlags uint16

//goland:noinspection GoUnusedConst
const (
	MonitorNone MonitorFlags = 1 << iota
	MonitorAll
	MonitorTicks
	MonitorBars
	MonitorEquity
	MonitorBalance
	MonitorPositionsOpened
	MonitorPositionsClosed
	MonitorPositionsPnLUpdated
	MonitorOrders
	MonitorOrdersRejected
	MonitorOrdersAccepted
	MonitorSignals
)

type Monitor struct {
	flags MonitorFlags
}

func NewMonitor(flags MonitorFlags) *Monitor {
	return &Monitor{
		flags: flags,
	}
}

func (m *Monitor) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(ctx context.Context, tick common.Tick) {
		if m.flags&MonitorTicks != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "tick", tick)
		}
		handler(ctx, tick)
	}
}

func (m *Monitor) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(ctx context.Context, bar common.Bar) {
		if m.flags&MonitorBars != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "bar", bar)
		}
		handler(ctx, bar)
	}
}

func (m *Monitor) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(ctx context.Context, equity common.Equity) {
		if m.flags&MonitorEquity != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "equity", equity)
		}
		handler(ctx, equity)
	}
}

func (m *Monitor) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(ctx context.Context, balance common.Balance) {
		if m.flags&MonitorBalance != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "balance", balance)
		}
		handler(ctx, balance)
	}
}

func (m *Monitor) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(ctx context.Context, position common.Position) {
		if m.flags&MonitorPositionsOpened != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_open", position)
		}
		handler(ctx, position)
	}
}

func (m *Monitor) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(ctx context.Context, position common.Position) {
		if m.flags&MonitorPositionsClosed != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_closed", position)
		}
		handler(ctx, position)
	}
}

func (m *Monitor) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(ctx context.Context, position common.Position) {
		if m.flags&MonitorPositionsPnLUpdated != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_update", position)
		}
		handler(ctx, position)
	}
}

func (m *Monitor) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(ctx context.Context, order common.Order) {
		if m.flags&MonitorOrders != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order", order)
		}
		handler(ctx, order)
	}
}

func (m *Monitor) WithOrderRejected(handler bus.OrderRejectedEventHandler) bus.OrderRejectedEventHandler {
	return func(ctx context.Context, rejected common.OrderRejected) {
		if m.flags&MonitorOrdersRejected != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order_rejected", rejected)
		}
		handler(ctx, rejected)
	}
}

func (m *Monitor) WithOrderAccepted(handler bus.OrderAcceptedEventHandler) bus.OrderAcceptedEventHandler {
	return func(ctx context.Context, accepted common.OrderAccepted) {
		if m.flags&MonitorOrdersAccepted != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order_accepted", accepted)
		}
		handler(ctx, accepted)
	}
}

func (m *Monitor) WithSignal(handler bus.SignalEventHandler) bus.SignalEventHandler {
	return func(ctx context.Context, signal common.Signal) {
		if m.flags&MonitorSignals != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "signal", signal)
		}
		handler(ctx, signal)
	}
}
