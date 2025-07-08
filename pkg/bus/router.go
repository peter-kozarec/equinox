package bus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type event struct {
	id   EventId
	data interface{}
}

type Router struct {
	// Channels
	done   chan error
	events chan event

	// Handlers
	TickHandler               TickEventHandler
	BarHandler                BarEventHandler
	EquityHandler             EquityEventHandler
	BalanceHandler            BalanceEventHandler
	PositionOpenedHandler     PositionOpenedEventHandler
	PositionClosedHandler     PositionClosedEventHandler
	PositionPnLUpdatedHandler PositionPnLUpdatedEventHandler
	OrderHandler              OrderEventHandler
	SignalHandler             SignalEventHandler

	// Statistics
	runTime       time.Duration
	postCount     uint64
	postFails     uint64
	dispatchCount uint64
	dispatchFails uint64
}

func NewRouter(eventCapacity int) *Router {
	return &Router{
		done:   make(chan error),
		events: make(chan event, eventCapacity),
	}
}

func (r *Router) Post(id EventId, data interface{}) error {
	select {
	case r.events <- event{id, data}:
		r.postCount++
		return nil
	default:
		r.postFails++
		return errors.New("event capacity reached")
	}
}

func (r *Router) Exec(ctx context.Context) {

	r.runTime = 0
	r.dispatchCount = 0
	r.dispatchFails = 0
	r.postCount = 0
	r.postFails = 0

	start := time.Now()
	defer func() {
		r.runTime += time.Since(start)
	}()

	for {
		select {
		case <-ctx.Done():
			r.done <- ctx.Err()
			return
		case ev := <-r.events:
			r.dispatchCount++
			if err := r.dispatch(ev); err != nil {
				r.dispatchFails++
				slog.Warn("dispatch failed", "error", err, "event", ev)
			}
		}
	}
}

func (r *Router) ExecLoop(ctx context.Context, doOnceCb func() error) {

	r.runTime = 0
	r.dispatchCount = 0
	r.dispatchFails = 0
	r.postCount = 0
	r.postFails = 0

	start := time.Now()
	defer func() {
		r.runTime += time.Since(start)
	}()

	for {
		select {
		case <-ctx.Done():
			r.done <- ctx.Err()
			return
		case ev := <-r.events:
			r.dispatchCount++
			if err := r.dispatch(ev); err != nil {
				r.dispatchFails++
				slog.Warn("dispatch failed", "error", err, "event", ev)
			}
		default:
			if err := doOnceCb(); err != nil {
				r.done <- err
				return
			}
		}
	}
}

func (r *Router) Done() <-chan error {
	return r.done
}

func (r *Router) PrintStatistics() {
	slog.Info("router statistics",
		"run_time", r.runTime,
		"post_count", r.postCount,
		"post_fails", r.postFails,
		"dispatch_count", r.dispatchCount,
		"dispatch_fails", r.dispatchFails,
		"throughput", float64(r.postCount)/r.runTime.Seconds())
}

func (r *Router) dispatch(ev event) error {
	switch ev.id {
	case TickEvent:
		tick, ok := ev.data.(common.Tick)
		if !ok {
			return errors.New("invalid type assertion for tick event")
		}
		if r.TickHandler != nil {
			r.TickHandler(tick)
		} else {
			slog.Debug("tick handler is nil")
		}
	case BarEvent:
		bar, ok := ev.data.(common.Bar)
		if !ok {
			return errors.New("invalid type assertion for bar event")
		}
		if r.BarHandler != nil {
			r.BarHandler(bar)
		} else {
			slog.Debug("bar handler is nil")
		}
	case EquityEvent:
		eq, ok := ev.data.(fixed.Point)
		if !ok {
			return errors.New("invalid type assertion for equity event")
		}
		if r.EquityHandler != nil {
			r.EquityHandler(eq)
		} else {
			slog.Debug("equity handler is nil")
		}
	case BalanceEvent:
		bal, ok := ev.data.(fixed.Point)
		if !ok {
			return errors.New("invalid type assertion for balance event")
		}
		if r.BalanceHandler != nil {
			r.BalanceHandler(bal)
		} else {
			slog.Debug("balance handler is nil")
		}
	case PositionOpenedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position opened event")
		}
		if r.PositionOpenedHandler != nil {
			r.PositionOpenedHandler(pos)
		} else {
			slog.Debug("position opened handler is nil")
		}
	case PositionClosedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position closed event")
		}
		if r.PositionClosedHandler != nil {
			r.PositionClosedHandler(pos)
		} else {
			slog.Debug("position closed handler is nil")
		}
	case PositionPnLUpdatedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position pnl updated event")
		}
		if r.PositionPnLUpdatedHandler != nil {
			r.PositionPnLUpdatedHandler(pos)
		} else {
			slog.Debug("position pnl updated handler is nil")
		}
	case OrderEvent:
		order, ok := ev.data.(common.Order)
		if !ok {
			return errors.New("invalid type assertion for order event")
		}
		if r.OrderHandler != nil {
			r.OrderHandler(order)
		} else {
			slog.Debug("order handler is nil")
		}
	case SignalEvent:
		sig, ok := ev.data.(common.Signal)
		if !ok {
			return errors.New("invalid type assertion for signal event")
		}
		if r.SignalHandler != nil {
			r.SignalHandler(sig)
		} else {
			slog.Debug("signal handler is nil")
		}
	default:
		return fmt.Errorf("unsupported event id: %v", ev.id)
	}
	return nil
}
