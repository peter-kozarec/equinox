package bus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"
)

type event struct {
	id   EventId
	data interface{}
}

type Router struct {
	events chan event

	TickHandler               TickEventHandler
	BarHandler                BarEventHandler
	EquityHandler             EquityEventHandler
	BalanceHandler            BalanceEventHandler
	PositionOpenedHandler     PositionOpenedEventHandler
	PositionClosedHandler     PositionClosedEventHandler
	PositionPnLUpdatedHandler PositionPnLUpdatedEventHandler
	OrderHandler              OrderEventHandler
	OrderAcceptedHandler      OrderAcceptedEventHandler
	OrderRejectedHandler      OrderRejectedEventHandler
	SignalHandler             SignalEventHandler

	runTime       time.Duration
	postCount     atomic.Uint64
	postFails     atomic.Uint64
	dispatchCount atomic.Uint64
	dispatchFails atomic.Uint64
}

func NewRouter(eventCapacity int) *Router {
	return &Router{
		events: make(chan event, eventCapacity),
	}
}

func (r *Router) Post(id EventId, data interface{}) error {
	select {
	case r.events <- event{id, data}:
		r.postCount.Add(1)
		return nil
	default:
		r.postFails.Add(1)
		return errors.New("event capacity reached")
	}
}

func (r *Router) Exec(ctx context.Context) <-chan error {

	r.runTime = 0
	r.dispatchCount.Store(0)
	r.dispatchFails.Store(0)
	r.postCount.Store(0)
	r.postFails.Store(0)

	start := time.Now()
	errChan := make(chan error)

	go func() {
		defer close(errChan)
		defer func() {
			r.runTime += time.Since(start)
		}()

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case ev := <-r.events:
				r.dispatchCount.Add(1)
				if err := r.dispatch(ctx, ev); err != nil {
					r.dispatchFails.Add(1)
					slog.Warn("dispatch failed", "error", err, "event", ev)
				}
			}
		}
	}()

	return errChan
}

func (r *Router) ExecLoop(ctx context.Context, doOnceCb func() error) <-chan error {

	r.runTime = 0
	r.dispatchCount.Store(0)
	r.dispatchFails.Store(0)
	r.postCount.Store(0)
	r.postFails.Store(0)

	start := time.Now()
	errChan := make(chan error)

	go func() {
		defer close(errChan)
		defer func() {
			r.runTime += time.Since(start)
		}()

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case ev := <-r.events:
				r.dispatchCount.Add(1)
				if err := r.dispatch(ctx, ev); err != nil {
					r.dispatchFails.Add(1)
					slog.Warn("dispatch failed", "error", err, "event", ev)
				}
			default:
				if err := doOnceCb(); err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	return errChan
}

func (r *Router) PrintStatistics() {
	runTimeSec := r.runTime.Seconds()

	postCount := r.postCount.Load()
	dispatchCount := r.dispatchCount.Load()
	postFails := r.postFails.Load()
	dispatchFails := r.dispatchFails.Load()

	throughput := 0.0
	if runTimeSec > 0 {
		throughput = float64(postCount) / runTimeSec
	}

	slog.Info("router statistics",
		"run_time", fmt.Sprintf("%.2fs", runTimeSec),
		"post_count", postCount,
		"post_fails", postFails,
		"dispatch_count", dispatchCount,
		"dispatch_fails", dispatchFails,
		"throughput", fmt.Sprintf("%.2f", throughput))
}

func (r *Router) dispatch(ctx context.Context, ev event) error {
	switch ev.id {
	case TickEvent:
		tick, ok := ev.data.(common.Tick)
		if !ok {
			return errors.New("invalid type assertion for tick event")
		}
		if r.TickHandler != nil {
			r.TickHandler(ctx, tick)
		} else {
			slog.Debug("tick handler is nil")
		}
	case BarEvent:
		bar, ok := ev.data.(common.Bar)
		if !ok {
			return errors.New("invalid type assertion for bar event")
		}
		if r.BarHandler != nil {
			r.BarHandler(ctx, bar)
		} else {
			slog.Debug("bar handler is nil")
		}
	case EquityEvent:
		eq, ok := ev.data.(common.Equity)
		if !ok {
			return errors.New("invalid type assertion for equity event")
		}
		if r.EquityHandler != nil {
			r.EquityHandler(ctx, eq)
		} else {
			slog.Debug("equity handler is nil")
		}
	case BalanceEvent:
		bal, ok := ev.data.(common.Balance)
		if !ok {
			return errors.New("invalid type assertion for balance event")
		}
		if r.BalanceHandler != nil {
			r.BalanceHandler(ctx, bal)
		} else {
			slog.Debug("balance handler is nil")
		}
	case PositionOpenedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position opened event")
		}
		if r.PositionOpenedHandler != nil {
			r.PositionOpenedHandler(ctx, pos)
		} else {
			slog.Debug("position opened handler is nil")
		}
	case PositionClosedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position closed event")
		}
		if r.PositionClosedHandler != nil {
			r.PositionClosedHandler(ctx, pos)
		} else {
			slog.Debug("position closed handler is nil")
		}
	case PositionPnLUpdatedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position pnl updated event")
		}
		if r.PositionPnLUpdatedHandler != nil {
			r.PositionPnLUpdatedHandler(ctx, pos)
		} else {
			slog.Debug("position pnl updated handler is nil")
		}
	case OrderEvent:
		order, ok := ev.data.(common.Order)
		if !ok {
			return errors.New("invalid type assertion for order event")
		}
		if r.OrderHandler != nil {
			r.OrderHandler(ctx, order)
		} else {
			slog.Debug("order handler is nil")
		}
	case OrderAcceptedEvent:
		orderAccepted, ok := ev.data.(common.OrderAccepted)
		if !ok {
			return errors.New("invalid type assertion for order accepted event")
		}
		if r.OrderAcceptedHandler != nil {
			r.OrderAcceptedHandler(ctx, orderAccepted)
		} else {
			slog.Debug("order accepted handler is nil")
		}
	case OrderRejectedEvent:
		orderRejected, ok := ev.data.(common.OrderRejected)
		if !ok {
			return errors.New("invalid type assertion for order rejected event")
		}
		if r.OrderRejectedHandler != nil {
			r.OrderRejectedHandler(ctx, orderRejected)
		}
	case SignalEvent:
		sig, ok := ev.data.(common.Signal)
		if !ok {
			return errors.New("invalid type assertion for signal event")
		}
		if r.SignalHandler != nil {
			r.SignalHandler(ctx, sig)
		} else {
			slog.Debug("signal handler is nil")
		}
	default:
		return fmt.Errorf("unsupported event id: %v", ev.id)
	}
	return nil
}
