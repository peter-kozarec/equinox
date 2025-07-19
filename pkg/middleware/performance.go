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

func (p *Performance) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(ctx context.Context, position common.Position) {
		startTime := time.Now()
		handler(ctx, position)
		p.totalPosOpenHandlerDur += time.Since(startTime)
		p.positionOpenedEventCounter++
	}
}

func (p *Performance) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(ctx context.Context, position common.Position) {
		startTime := time.Now()
		handler(ctx, position)
		p.totalPosClosHandlerDur += time.Since(startTime)
		p.positionClosedEventCounter++
	}
}

func (p *Performance) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
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

func (p *Performance) WithOrderRejected(handler bus.OrderRejectedEventHandler) bus.OrderRejectedEventHandler {
	return func(ctx context.Context, rejected common.OrderRejected) {
		startTime := time.Now()
		handler(ctx, rejected)
		p.totalOrderRejectedDur += time.Since(startTime)
		p.orderRejectedEventCounter++
	}
}

func (p *Performance) WithOrderAccepted(handler bus.OrderAcceptedEventHandler) bus.OrderAcceptedEventHandler {
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

func (p *Performance) PrintStatistics() {

	var fields []slog.Attr

	// Tick events
	if p.tickEventCounter > 0 {
		avgTick := p.totalTickHandlerDur / time.Duration(p.tickEventCounter)
		if avgTick > 0 {
			fields = append(fields,
				slog.Int64("tick_event_count", p.tickEventCounter),
				slog.String("tick_avg_duration", fmt.Sprintf("%dns", avgTick.Nanoseconds())),
				slog.String("tick_total_duration", fmt.Sprintf("%dns", p.totalTickHandlerDur.Nanoseconds())),
			)
		}
	}

	// Bar events
	if p.barEventCounter > 0 {
		avgBar := p.totalBarHandlerDur / time.Duration(p.barEventCounter)
		if avgBar > 0 {
			fields = append(fields,
				slog.Int64("bar_event_count", p.barEventCounter),
				slog.String("bar_avg_duration", fmt.Sprintf("%dns", avgBar.Nanoseconds())),
				slog.String("bar_total_duration", fmt.Sprintf("%dns", p.totalBarHandlerDur.Nanoseconds())),
			)
		}
	}

	// Balance events
	if p.balanceEventCounter > 0 {
		avgBalance := p.totalBalanceHandlerDur / time.Duration(p.balanceEventCounter)
		if avgBalance > 0 {
			fields = append(fields,
				slog.Int64("balance_event_count", p.balanceEventCounter),
				slog.String("balance_avg_duration", fmt.Sprintf("%dns", avgBalance.Nanoseconds())),
				slog.String("balance_total_duration", fmt.Sprintf("%dns", p.totalBalanceHandlerDur.Nanoseconds())),
			)
		}
	}

	// Equity events
	if p.equityEventCounter > 0 {
		avgEquity := p.totalEquityHandlerDur / time.Duration(p.equityEventCounter)
		if avgEquity > 0 {
			fields = append(fields,
				slog.Int64("equity_event_count", p.equityEventCounter),
				slog.String("equity_avg_duration", fmt.Sprintf("%dns", avgEquity.Nanoseconds())),
				slog.String("equity_total_duration", fmt.Sprintf("%dns", p.totalEquityHandlerDur.Nanoseconds())),
			)
		}
	}

	// Position opened events
	if p.positionOpenedEventCounter > 0 {
		avgPosOpen := p.totalPosOpenHandlerDur / time.Duration(p.positionOpenedEventCounter)
		if avgPosOpen > 0 {
			fields = append(fields,
				slog.Int64("position_open_event_count", p.positionOpenedEventCounter),
				slog.String("position_open_avg_duration", fmt.Sprintf("%dns", avgPosOpen.Nanoseconds())),
				slog.String("position_open_total_duration", fmt.Sprintf("%dns", p.totalPosOpenHandlerDur.Nanoseconds())),
			)
		}
	}

	// Position closed events
	if p.positionClosedEventCounter > 0 {
		avgPosClosed := p.totalPosClosHandlerDur / time.Duration(p.positionClosedEventCounter)
		if avgPosClosed > 0 {
			fields = append(fields,
				slog.Int64("position_closed_event_count", p.positionClosedEventCounter),
				slog.String("position_closed_avg_duration", fmt.Sprintf("%dns", avgPosClosed.Nanoseconds())),
				slog.String("position_closed_total_duration", fmt.Sprintf("%dns", p.totalPosClosHandlerDur.Nanoseconds())),
			)
		}
	}

	// Position PnL updated events
	if p.positionPnLUpdatedEventCounter > 0 {
		avgPosPnlUpd := p.totalPosUpdtHandlerDur / time.Duration(p.positionPnLUpdatedEventCounter)
		if avgPosPnlUpd > 0 {
			fields = append(fields,
				slog.Int64("position_update_event_count", p.positionPnLUpdatedEventCounter),
				slog.String("position_update_avg_duration", fmt.Sprintf("%dns", avgPosPnlUpd.Nanoseconds())),
				slog.String("position_update_total_duration", fmt.Sprintf("%dns", p.totalPosUpdtHandlerDur.Nanoseconds())),
			)
		}
	}

	// Order events
	if p.orderEventCounter > 0 {
		avgOrder := p.totalOrderHandlerDur / time.Duration(p.orderEventCounter)
		if avgOrder > 0 {
			fields = append(fields,
				slog.Int64("order_event_count", p.orderEventCounter),
				slog.String("order_avg_duration", fmt.Sprintf("%dns", avgOrder.Nanoseconds())),
				slog.String("order_total_duration", fmt.Sprintf("%dns", p.totalOrderHandlerDur.Nanoseconds())),
			)
		}
	}

	// Order rejected events
	if p.orderRejectedEventCounter > 0 {
		avgOrderRejected := p.totalOrderRejectedDur / time.Duration(p.orderRejectedEventCounter)
		if avgOrderRejected > 0 {
			fields = append(fields,
				slog.Int64("order_rejected_event_count", p.orderRejectedEventCounter),
				slog.String("order_rejected_avg_duration", fmt.Sprintf("%dns", avgOrderRejected.Nanoseconds())),
				slog.String("order_rejected_total_duration", fmt.Sprintf("%dns", p.totalOrderRejectedDur.Nanoseconds())),
			)
		}
	}

	// Order accepted events
	if p.orderAcceptedEventCounter > 0 {
		avgOrderAccepted := p.totalOrderAcceptedDur / time.Duration(p.orderAcceptedEventCounter)
		if avgOrderAccepted > 0 {
			fields = append(fields,
				slog.Int64("order_accepted_event_count", p.orderAcceptedEventCounter),
				slog.String("order_accepted_avg_duration", fmt.Sprintf("%dns", avgOrderAccepted.Nanoseconds())),
				slog.String("order_accepted_total_duration", fmt.Sprintf("%dns", p.totalOrderAcceptedDur.Nanoseconds())),
			)
		}
	}

	// Signal events
	if p.signalEventCounter > 0 {
		avgSignal := p.totalSignalHandlerDur / time.Duration(p.signalEventCounter)
		if avgSignal > 0 {
			fields = append(fields,
				slog.Int64("signal_event_count", p.signalEventCounter),
				slog.String("signal_avg_duration", fmt.Sprintf("%dns", avgSignal.Nanoseconds())),
				slog.String("signal_total_duration", fmt.Sprintf("%dns", p.totalSignalHandlerDur.Nanoseconds())))
		}
	}

	slog.LogAttrs(context.Background(), slog.LevelInfo, "performance statistics", fields...)
}
