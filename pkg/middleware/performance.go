package middleware

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type Performance struct {
	logger *zap.Logger

	totalTickHandlerDur    time.Duration
	totalBarHandlerDur     time.Duration
	totalBalanceHandlerDur time.Duration
	totalEquityHandlerDur  time.Duration
	totalPosOpenHandlerDur time.Duration
	totalPosUpdtHandlerDur time.Duration
	totalPosClosHandlerDur time.Duration
	totalOrderHandlerDur   time.Duration
	totalSignalHandlerDur  time.Duration
}

func NewPerformance(logger *zap.Logger) *Performance {
	return &Performance{
		logger: logger,
	}
}

func (p *Performance) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(tick common.Tick) {
		startTime := time.Now()
		handler(tick) // Call the original handler
		p.totalTickHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar common.Bar) {
		startTime := time.Now()
		handler(bar) // Call the original handler
		p.totalBarHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance fixed.Point) {
		startTime := time.Now()
		handler(balance) // Call the original handler
		p.totalBalanceHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) WithEquity(handler bus.EquityEventHandler) bus.EquityEventHandler {
	return func(equity fixed.Point) {
		startTime := time.Now()
		handler(equity) // Call the original handler
		p.totalEquityHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) WithPositionOpened(handler bus.PositionOpenedEventHandler) bus.PositionOpenedEventHandler {
	return func(position common.Position) {
		startTime := time.Now()
		handler(position) // Call the original handler
		p.totalPosOpenHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position common.Position) {
		startTime := time.Now()
		handler(position) // Call the original handler
		p.totalPosClosHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) WithPositionPnLUpdated(handler bus.PositionPnLUpdatedEventHandler) bus.PositionPnLUpdatedEventHandler {
	return func(position common.Position) {
		startTime := time.Now()
		handler(position) // Call the original handler
		p.totalPosUpdtHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) WithOrder(handler bus.OrderEventHandler) bus.OrderEventHandler {
	return func(order common.Order) {
		startTime := time.Now()
		handler(order) // Call the original handler
		p.totalOrderHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) WithSignal(handler bus.SignalEventHandler) bus.SignalEventHandler {
	return func(signal common.Signal) {
		startTime := time.Now()
		handler(signal) // Call the original handler
		p.totalSignalHandlerDur += time.Since(startTime)
	}
}

func (p *Performance) PrintStatistics(t *Telemetry) {
	if t == nil {
		p.logger.Warn("Telemetry is nil; cannot compute performance statistics")
		return
	}

	var fields []zap.Field

	// Tick events
	if t.tickEventCounter > 0 {
		avgTick := p.totalTickHandlerDur / time.Duration(t.tickEventCounter)
		if avgTick > 0 {
			fields = append(fields,
				zap.Duration("tick_avg_duration", avgTick),
				zap.Duration("tick_total_duration", p.totalTickHandlerDur),
			)
		}
	}

	// Bar events
	if t.barEventCounter > 0 {
		avgBar := p.totalBarHandlerDur / time.Duration(t.barEventCounter)
		if avgBar > 0 {
			fields = append(fields,
				zap.Duration("bar_avg_duration", avgBar),
				zap.Duration("bar_total_duration", p.totalBarHandlerDur),
			)
		}
	}

	// Balance events
	if t.balanceEventCounter > 0 {
		avgBalance := p.totalBalanceHandlerDur / time.Duration(t.balanceEventCounter)
		if avgBalance > 0 {
			fields = append(fields,
				zap.Duration("balance_avg_duration", avgBalance),
				zap.Duration("balance_total_duration", p.totalBalanceHandlerDur),
			)
		}
	}

	// Equity events
	if t.equityEventCounter > 0 {
		avgEquity := p.totalEquityHandlerDur / time.Duration(t.equityEventCounter)
		if avgEquity > 0 {
			fields = append(fields,
				zap.Duration("equity_avg_duration", avgEquity),
				zap.Duration("equity_total_duration", p.totalEquityHandlerDur),
			)
		}
	}

	// Position opened events
	if t.positionOpenedEventCounter > 0 {
		avgPosOpen := p.totalPosOpenHandlerDur / time.Duration(t.positionOpenedEventCounter)
		if avgPosOpen > 0 {
			fields = append(fields,
				zap.Duration("position_open_avg_duration", avgPosOpen),
				zap.Duration("position_open_total_duration", p.totalPosOpenHandlerDur),
			)
		}
	}

	// Position closed events
	if t.positionClosedEventCounter > 0 {
		avgPosClosed := p.totalPosClosHandlerDur / time.Duration(t.positionClosedEventCounter)
		if avgPosClosed > 0 {
			fields = append(fields,
				zap.Duration("position_closed_avg_duration", avgPosClosed),
				zap.Duration("position_closed_total_duration", p.totalPosClosHandlerDur),
			)
		}
	}

	// Position PnL updated events
	if t.positionPnLUpdatedEventCounter > 0 {
		avgPosPnlUpd := p.totalPosUpdtHandlerDur / time.Duration(t.positionPnLUpdatedEventCounter)
		if avgPosPnlUpd > 0 {
			fields = append(fields,
				zap.Duration("position_pnl_update_avg_duration", avgPosPnlUpd),
				zap.Duration("position_pnl_update_total_duration", p.totalPosUpdtHandlerDur),
			)
		}
	}

	// Order events
	if t.orderEventCounter > 0 {
		avgOrder := p.totalOrderHandlerDur / time.Duration(t.orderEventCounter)
		if avgOrder > 0 {
			fields = append(fields,
				zap.Duration("order_avg_duration", avgOrder),
				zap.Duration("order_total_duration", p.totalOrderHandlerDur),
			)
		}
	}

	// Signal events
	if t.signalEventCounter > 0 {
		avgSignal := p.totalSignalHandlerDur / time.Duration(t.signalEventCounter)
		if avgSignal > 0 {
			fields = append(fields,
				zap.Duration("signal_avg_duration", avgSignal),
				zap.Duration("signal_total_duration", p.totalSignalHandlerDur))
		}
	}

	p.logger.Info("performance statistics", fields...)
}
