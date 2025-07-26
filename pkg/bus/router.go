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

	OnTick             TickEventHandler
	OnBar              BarEventHandler
	OnEquity           EquityEventHandler
	OnBalance          BalanceEventHandler
	OnPositionOpen     PositionOpenEventHandler
	OnPositionClose    PositionCloseEventHandler
	OnPositionUpdate   PositionUpdateEventHandler
	OnOrder            OrderEventHandler
	OnOrderAcceptance  OrderAcceptanceHandler
	OnOrderRejection   OrderRejectionEventHandler
	OnSignal           SignalEventHandler
	OnSignalAcceptance SignalAcceptanceEventHandler
	OnSignalRejection  SignalRejectionEventHandler

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

func (r *Router) DrainEvents(ctx context.Context) error {
	for {
		select {
		case ev := <-r.events:
			if r := r.dispatch(ctx, ev); r != nil {
				return fmt.Errorf("dispatch failed: %w", r)
			}
		default:
			return nil
		}
	}
}

func (r *Router) GetStatistics() Statistics {
	stats := Statistics{}

	stats.RunTime = r.runTime
	stats.PostCount = r.postCount.Load()
	stats.DispatchCount = r.dispatchCount.Load()
	stats.PostFails = r.postFails.Load()
	stats.DispatchFails = r.dispatchFails.Load()

	if stats.RunTime > 0 {
		stats.Throughput = float64(stats.PostCount) / stats.RunTime.Seconds()
	}

	return stats
}

func (r *Router) dispatch(ctx context.Context, ev event) error {
	switch ev.id {
	case TickEvent:
		tick, ok := ev.data.(common.Tick)
		if !ok {
			return errors.New("invalid type assertion for tick event")
		}
		if r.OnTick != nil {
			r.OnTick(ctx, tick)
		} else {
			slog.Debug("tick handler is nil")
		}
	case BarEvent:
		bar, ok := ev.data.(common.Bar)
		if !ok {
			return errors.New("invalid type assertion for bar event")
		}
		if r.OnBar != nil {
			r.OnBar(ctx, bar)
		} else {
			slog.Debug("bar handler is nil")
		}
	case EquityEvent:
		eq, ok := ev.data.(common.Equity)
		if !ok {
			return errors.New("invalid type assertion for equity event")
		}
		if r.OnEquity != nil {
			r.OnEquity(ctx, eq)
		} else {
			slog.Debug("equity handler is nil")
		}
	case BalanceEvent:
		bal, ok := ev.data.(common.Balance)
		if !ok {
			return errors.New("invalid type assertion for balance event")
		}
		if r.OnBalance != nil {
			r.OnBalance(ctx, bal)
		} else {
			slog.Debug("balance handler is nil")
		}
	case PositionOpenEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position opened event")
		}
		if r.OnPositionOpen != nil {
			r.OnPositionOpen(ctx, pos)
		} else {
			slog.Debug("position opened handler is nil")
		}
	case PositionCloseEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position closed event")
		}
		if r.OnPositionClose != nil {
			r.OnPositionClose(ctx, pos)
		} else {
			slog.Debug("position closed handler is nil")
		}
	case PositionUpdateEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position pnl updated event")
		}
		if r.OnPositionUpdate != nil {
			r.OnPositionUpdate(ctx, pos)
		} else {
			slog.Debug("position pnl updated handler is nil")
		}
	case OrderEvent:
		order, ok := ev.data.(common.Order)
		if !ok {
			return errors.New("invalid type assertion for order event")
		}
		if r.OnOrder != nil {
			r.OnOrder(ctx, order)
		} else {
			slog.Debug("order handler is nil")
		}
	case OrderAcceptanceEvent:
		orderAccepted, ok := ev.data.(common.OrderAccepted)
		if !ok {
			return errors.New("invalid type assertion for order accepted event")
		}
		if r.OnOrderAcceptance != nil {
			r.OnOrderAcceptance(ctx, orderAccepted)
		} else {
			slog.Debug("order accepted handler is nil")
		}
	case OrderRejectionEvent:
		orderRejected, ok := ev.data.(common.OrderRejected)
		if !ok {
			return errors.New("invalid type assertion for order rejected event")
		}
		if r.OnOrderRejection != nil {
			r.OnOrderRejection(ctx, orderRejected)
		} else {
			slog.Debug("order rejected handler is nil")
		}
	case SignalEvent:
		sig, ok := ev.data.(common.Signal)
		if !ok {
			return errors.New("invalid type assertion for signal event")
		}
		if r.OnSignal != nil {
			r.OnSignal(ctx, sig)
		} else {
			slog.Debug("signal handler is nil")
		}
	case SignalAcceptanceEvent:
		sigAccepted, ok := ev.data.(common.SignalAccepted)
		if !ok {
			return errors.New("invalid type assertion for signal accepted event")
		}
		if r.OnSignalAcceptance != nil {
			r.OnSignalAcceptance(ctx, sigAccepted)
		} else {
			slog.Debug("signal accepted handler is nil")
		}
	case SignalRejectionEvent:
		sigRejected, ok := ev.data.(common.SignalRejected)
		if !ok {
			return errors.New("invalid type assertion for signal rejected event")
		}
		if r.OnSignalRejection != nil {
			r.OnSignalRejection(ctx, sigRejected)
		} else {
			slog.Debug("signal rejected handler is nil")
		}
	default:
		return fmt.Errorf("unsupported event id: %v", ev.id)
	}
	return nil
}
