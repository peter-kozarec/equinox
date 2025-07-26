package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
)

type Performance struct {
	tickEventCounter               int64
	barEventCounter                int64
	balanceEventCounter            int64
	equityEventCounter             int64
	positionOpenedEventCounter     int64
	positionClosedEventCounter     int64
	positionPnLUpdatedEventCounter int64
	orderEventCounter              int64
	orderRejectedEventCounter      int64
	orderAcceptedEventCounter      int64
	signalEventCounter             int64
	signalRejectedEventCounter     int64
	signalAcceptedEventCounter     int64

	totalTickHandlerDur    time.Duration
	totalBarHandlerDur     time.Duration
	totalBalanceHandlerDur time.Duration
	totalEquityHandlerDur  time.Duration
	totalPosOpenHandlerDur time.Duration
	totalPosUpdtHandlerDur time.Duration
	totalPosClosHandlerDur time.Duration
	totalOrderHandlerDur   time.Duration
	totalOrderRejectedDur  time.Duration
	totalOrderAcceptedDur  time.Duration
	totalSignalHandlerDur  time.Duration
	totalSignalRejectedDur time.Duration
	totalSignalAcceptedDur time.Duration
}

func NewPerformance() *Performance {
	return &Performance{}
}

func (p *Performance) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(ctx context.Context, tick common.Tick) {
		startTime := time.Now()
		handler(ctx, tick)
		p.totalTickHandlerDur += time.Since(startTime)
		p.tickEventCounter++
	}
}

func (p *Performance) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(ctx context.Context, bar common.Bar) {
		startTime := time.Now()
		handler(ctx, bar)
		p.totalBarHandlerDur += time.Since(startTime)
		p.barEventCounter++
	}
}

func (p *Performance) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(ctx context.Context, balance common.Balance) {
		startTime := time.Now()
		handler(ctx, balance)
		p.totalBalanceHandlerDur += time.Since(startTime)
		p.balanceEventCounter++
	}
}

func (p *Performance) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(ctx context.Context, equity common.Equity) {
		startTime := time.Now()
		handler(ctx, equity)
		p.totalEquityHandlerDur += time.Since(startTime)
		p.equityEventCounter++
	}
}

func (p *Performance) WithPositionOpen(handler bus.PositionOpenEventHandler) bus.PositionOpenEventHandler {
	return func(ctx context.Context, position common.Position) {
		startTime := time.Now()
		handler(ctx, position)
		p.totalPosOpenHandlerDur += time.Since(startTime)
		p.positionOpenedEventCounter++
	}
}

func (p *Performance) WithPositionClose(handler bus.PositionCloseEventHandler) bus.PositionCloseEventHandler {
	return func(ctx context.Context, position common.Position) {
		startTime := time.Now()
		handler(ctx, position)
		p.totalPosClosHandlerDur += time.Since(startTime)
		p.positionClosedEventCounter++
	}
}

func (p *Performance) WithPositionUpdate(handler bus.PositionUpdateEventHandler) bus.PositionUpdateEventHandler {
	return func(ctx context.Context, position common.Position) {
		startTime := time.Now()
		handler(ctx, position)
		p.totalPosUpdtHandlerDur += time.Since(startTime)
		p.positionPnLUpdatedEventCounter++
	}
}

func (p *Performance) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(ctx context.Context, order common.Order) {
		startTime := time.Now()
		handler(ctx, order)
		p.totalOrderHandlerDur += time.Since(startTime)
		p.orderEventCounter++
	}
}

func (p *Performance) WithOrderRejection(handler bus.OrderRejectionEventHandler) bus.OrderRejectionEventHandler {
	return func(ctx context.Context, rejected common.OrderRejected) {
		startTime := time.Now()
		handler(ctx, rejected)
		p.totalOrderRejectedDur += time.Since(startTime)
		p.orderRejectedEventCounter++
	}
}

func (p *Performance) WithOrderAcceptance(handler bus.OrderAcceptanceHandler) bus.OrderAcceptanceHandler {
	return func(ctx context.Context, accepted common.OrderAccepted) {
		startTime := time.Now()
		handler(ctx, accepted)
		p.totalOrderAcceptedDur += time.Since(startTime)
		p.orderAcceptedEventCounter++
	}
}

func (p *Performance) WithSignal(handler bus.SignalEventHandler) bus.SignalEventHandler {
	return func(ctx context.Context, signal common.Signal) {
		startTime := time.Now()
		handler(ctx, signal)
		p.totalSignalHandlerDur += time.Since(startTime)
		p.signalEventCounter++
	}
}

func (p *Performance) WithSignalRejection(handler bus.SignalRejectionEventHandler) bus.SignalRejectionEventHandler {
	return func(ctx context.Context, rejected common.SignalRejected) {
		startTime := time.Now()
		handler(ctx, rejected)
		p.totalSignalRejectedDur += time.Since(startTime)
		p.signalRejectedEventCounter++
	}
}

func (p *Performance) WithSignalAcceptance(handler bus.SignalAcceptanceEventHandler) bus.SignalAcceptanceEventHandler {
	return func(ctx context.Context, accepted common.SignalAccepted) {
		startTime := time.Now()
		handler(ctx, accepted)
		p.totalSignalAcceptedDur += time.Since(startTime)
		p.signalAcceptedEventCounter++
	}
}

func (p *Performance) PrintStatistics() {
	var args []any

	if p.tickEventCounter > 0 {
		avgTick := p.totalTickHandlerDur / time.Duration(p.tickEventCounter)
		if avgTick > 0 {
			args = append(args,
				"tick_event_count", p.tickEventCounter,
				"tick_avg_duration", fmt.Sprintf("%dns", avgTick.Nanoseconds()),
			)
		}
	}

	if p.barEventCounter > 0 {
		avgBar := p.totalBarHandlerDur / time.Duration(p.barEventCounter)
		if avgBar > 0 {
			args = append(args,
				"bar_event_count", p.barEventCounter,
				"bar_avg_duration", fmt.Sprintf("%dns", avgBar.Nanoseconds()),
			)
		}
	}

	if p.balanceEventCounter > 0 {
		avgBalance := p.totalBalanceHandlerDur / time.Duration(p.balanceEventCounter)
		if avgBalance > 0 {
			args = append(args,
				"balance_event_count", p.balanceEventCounter,
				"balance_avg_duration", fmt.Sprintf("%dns", avgBalance.Nanoseconds()),
			)
		}
	}

	if p.equityEventCounter > 0 {
		avgEquity := p.totalEquityHandlerDur / time.Duration(p.equityEventCounter)
		if avgEquity > 0 {
			args = append(args,
				"equity_event_count", p.equityEventCounter,
				"equity_avg_duration", fmt.Sprintf("%dns", avgEquity.Nanoseconds()),
			)
		}
	}

	if p.positionOpenedEventCounter > 0 {
		avgPosOpen := p.totalPosOpenHandlerDur / time.Duration(p.positionOpenedEventCounter)
		if avgPosOpen > 0 {
			args = append(args,
				"position_open_event_count", p.positionOpenedEventCounter,
				"position_open_avg_duration", fmt.Sprintf("%dns", avgPosOpen.Nanoseconds()),
			)
		}
	}

	if p.positionClosedEventCounter > 0 {
		avgPosClosed := p.totalPosClosHandlerDur / time.Duration(p.positionClosedEventCounter)
		if avgPosClosed > 0 {
			args = append(args,
				"position_closed_event_count", p.positionClosedEventCounter,
				"position_closed_avg_duration", fmt.Sprintf("%dns", avgPosClosed.Nanoseconds()),
			)
		}
	}

	if p.positionPnLUpdatedEventCounter > 0 {
		avgPosPnlUpd := p.totalPosUpdtHandlerDur / time.Duration(p.positionPnLUpdatedEventCounter)
		if avgPosPnlUpd > 0 {
			args = append(args,
				"position_update_event_count", p.positionPnLUpdatedEventCounter,
				"position_update_avg_duration", fmt.Sprintf("%dns", avgPosPnlUpd.Nanoseconds()),
			)
		}
	}

	if p.orderEventCounter > 0 {
		avgOrder := p.totalOrderHandlerDur / time.Duration(p.orderEventCounter)
		if avgOrder > 0 {
			args = append(args,
				"order_event_count", p.orderEventCounter,
				"order_avg_duration", fmt.Sprintf("%dns", avgOrder.Nanoseconds()),
			)
		}
	}

	if p.orderRejectedEventCounter > 0 {
		avgOrderRejected := p.totalOrderRejectedDur / time.Duration(p.orderRejectedEventCounter)
		if avgOrderRejected > 0 {
			args = append(args,
				"order_rejected_event_count", p.orderRejectedEventCounter,
				"order_rejected_avg_duration", fmt.Sprintf("%dns", avgOrderRejected.Nanoseconds()),
			)
		}
	}

	if p.orderAcceptedEventCounter > 0 {
		avgOrderAccepted := p.totalOrderAcceptedDur / time.Duration(p.orderAcceptedEventCounter)
		if avgOrderAccepted > 0 {
			args = append(args,
				"order_accepted_event_count", p.orderAcceptedEventCounter,
				"order_accepted_avg_duration", fmt.Sprintf("%dns", avgOrderAccepted.Nanoseconds()),
			)
		}
	}

	if p.signalEventCounter > 0 {
		avgSignal := p.totalSignalHandlerDur / time.Duration(p.signalEventCounter)
		if avgSignal > 0 {
			args = append(args,
				"signal_event_count", p.signalEventCounter,
				"signal_avg_duration", fmt.Sprintf("%dns", avgSignal.Nanoseconds()),
			)
		}
	}

	if p.signalRejectedEventCounter > 0 {
		avgSignalRejected := p.totalSignalRejectedDur / time.Duration(p.signalRejectedEventCounter)
		if avgSignalRejected > 0 {
			args = append(args,
				"signal_rejected_event_count", p.signalRejectedEventCounter,
				"signal_rejected_avg_duration", fmt.Sprintf("%dns", avgSignalRejected.Nanoseconds()),
			)
		}
	}

	if p.signalAcceptedEventCounter > 0 {
		avgSignalAccepted := p.totalSignalAcceptedDur / time.Duration(p.signalAcceptedEventCounter)
		if avgSignalAccepted > 0 {
			args = append(args,
				"signal_accepted_event_count", p.signalAcceptedEventCounter,
				"signal_accepted_avg_duration", fmt.Sprintf("%dns", avgSignalAccepted.Nanoseconds()),
			)
		}
	}

	if len(args) > 0 {
		slog.Info("performance statistics", args...)
	}
}
