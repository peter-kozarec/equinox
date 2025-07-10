package middleware

import (
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
	return func(tick common.Tick) {
		if m.flags&MonitorTicks != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "tick", tick)
		}
		handler(tick)
	}
}

func (m *Monitor) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar common.Bar) {
		if m.flags&MonitorBars != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "bar", bar)
		}
		handler(bar)
	}
}

func (m *Monitor) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity common.Equity) {
		if m.flags&MonitorEquity != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "equity", equity)
		}
		handler(equity)
	}
}

func (m *Monitor) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance common.Balance) {
		if m.flags&MonitorBalance != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "balance", balance)
		}
		handler(balance)
	}
}

func (m *Monitor) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(position common.Position) {
		if m.flags&MonitorPositionsOpened != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_open", position)
		}
		handler(position)
	}
}

func (m *Monitor) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position common.Position) {
		if m.flags&MonitorPositionsClosed != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_closed", position)
		}
		handler(position)
	}
}

func (m *Monitor) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(position common.Position) {
		if m.flags&MonitorPositionsPnLUpdated != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "position_update", position)
		}
		handler(position)
	}
}

func (m *Monitor) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(order common.Order) {
		if m.flags&MonitorOrders != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order", order)
		}
		handler(order)
	}
}

func (m *Monitor) WithOrderRejected(handler bus.OrderRejectedEventHandler) bus.OrderRejectedEventHandler {
	return func(rejected common.OrderRejected) {
		if m.flags&MonitorOrdersRejected != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order_rejected", rejected)
		}
		handler(rejected)
	}
}

func (m *Monitor) WithOrderAccepted(handler bus.OrderAcceptedEventHandler) bus.OrderAcceptedEventHandler {
	return func(accepted common.OrderAccepted) {
		if m.flags&MonitorOrdersAccepted != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "order_accepted", accepted)
		}
		handler(accepted)
	}
}

func (m *Monitor) WithSignal(handler bus.SignalEventHandler) bus.SignalEventHandler {
	return func(signal common.Signal) {
		if m.flags&MonitorSignals != 0 || m.flags&MonitorAll != 0 {
			slog.Info("event", "signal", signal)
		}
		handler(signal)
	}
}
