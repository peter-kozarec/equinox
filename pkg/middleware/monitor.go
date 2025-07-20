package middleware

import (
	"context"
	"log/slog"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
)

type MonitorFlags uint16

const (
	MonitorNone MonitorFlags = 1 << iota
	MonitorAll
	MonitorTick
	MonitorBar
	MonitorEquity
	MonitorBalance
	MonitorPositionOpen
	MonitorPositionClose
	MonitorPositionUpdate
	MonitorOrder
	MonitorOrderRejection
	MonitorOrderAcceptance
	MonitorSignal
	MonitorSignalRejection
	MonitorSignalAcceptance
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
		if m.flags&MonitorTick != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "tick", tick)
		}
		handler(ctx, tick)
	}
}

func (m *Monitor) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(ctx context.Context, bar common.Bar) {
		if m.flags&MonitorBar != 0 || m.flags&MonitorAll != 0 {
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

func (m *Monitor) WithPositionOpen(handler bus.PositionOpenEventHandler) bus.PositionOpenEventHandler {
	return func(ctx context.Context, position common.Position) {
		if m.flags&MonitorPositionOpen != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_open", position)
		}
		handler(ctx, position)
	}
}

func (m *Monitor) WithPositionClose(handler bus.PositionCloseEventHandler) bus.PositionCloseEventHandler {
	return func(ctx context.Context, position common.Position) {
		if m.flags&MonitorPositionClose != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_closed", position)
		}
		handler(ctx, position)
	}
}

func (m *Monitor) WithPositionUpdate(handler bus.PositionUpdateEventHandler) bus.PositionUpdateEventHandler {
	return func(ctx context.Context, position common.Position) {
		if m.flags&MonitorPositionUpdate != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_update", position)
		}
		handler(ctx, position)
	}
}

func (m *Monitor) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(ctx context.Context, order common.Order) {
		if m.flags&MonitorOrder != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order", order)
		}
		handler(ctx, order)
	}
}

func (m *Monitor) WithOrderRejection(handler bus.OrderRejectionEventHandler) bus.OrderRejectionEventHandler {
	return func(ctx context.Context, rejected common.OrderRejected) {
		if m.flags&MonitorOrderRejection != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order_rejected", rejected)
		}
		handler(ctx, rejected)
	}
}

func (m *Monitor) WithOrderAcceptance(handler bus.OrderAcceptanceHandler) bus.OrderAcceptanceHandler {
	return func(ctx context.Context, accepted common.OrderAccepted) {
		if m.flags&MonitorOrderAcceptance != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order_accepted", accepted)
		}
		handler(ctx, accepted)
	}
}

func (m *Monitor) WithSignal(handler bus.SignalEventHandler) bus.SignalEventHandler {
	return func(ctx context.Context, signal common.Signal) {
		if m.flags&MonitorSignal != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "signal", signal)
		}
		handler(ctx, signal)
	}
}

func (m *Monitor) WithSignalRejection(handler bus.SignalRejectionEventHandler) bus.SignalRejectionEventHandler {
	return func(ctx context.Context, rejected common.SignalRejected) {
		if m.flags&MonitorSignalRejection != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "signal_rejected", rejected)
		}
		handler(ctx, rejected)
	}
}

func (m *Monitor) WithSignalAcceptance(handler bus.SignalAcceptanceEventHandler) bus.SignalAcceptanceEventHandler {
	return func(ctx context.Context, accepted common.SignalAccepted) {
		if m.flags&MonitorSignalAcceptance != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "signal_accepted", accepted)
		}
		handler(ctx, accepted)
	}
}
