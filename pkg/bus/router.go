package bus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type event struct {
	id   EventId
	data interface{}
}

type Router struct {
	logger *zap.Logger

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

func NewRouter(logger *zap.Logger, eventCapacity int) *Router {
	return &Router{
		logger: logger,
		done:   make(chan error),
		events: make(chan event, eventCapacity),
	}
}

func (router *Router) Post(id EventId, data interface{}) error {
	select {
	case router.events <- event{id, data}:
		router.postCount++
		return nil
	default:
		router.postFails++
		return errors.New("event capacity reached")
	}
}

func (router *Router) Exec(ctx context.Context) {

	router.runTime = 0
	router.dispatchCount = 0
	router.dispatchFails = 0
	router.postCount = 0
	router.postFails = 0

	start := time.Now()
	defer func() {
		router.runTime += time.Since(start)
	}()

	for {
		select {
		case <-ctx.Done():
			router.done <- ctx.Err()
			return
		case ev := <-router.events:
			router.dispatchCount++
			if err := router.dispatch(ev); err != nil {
				router.dispatchFails++
				router.logger.Warn("dispatch failed",
					zap.Error(err),
					zap.Any("event", ev))
			}
		}
	}
}

func (router *Router) ExecLoop(ctx context.Context, doOnceCb func() error) {

	router.runTime = 0
	router.dispatchCount = 0
	router.dispatchFails = 0
	router.postCount = 0
	router.postFails = 0

	start := time.Now()
	defer func() {
		router.runTime += time.Since(start)
	}()

	for {
		select {
		case <-ctx.Done():
			router.done <- ctx.Err()
			return
		case ev := <-router.events:
			router.dispatchCount++
			if err := router.dispatch(ev); err != nil {
				router.dispatchFails++
				router.logger.Warn("dispatch failed",
					zap.Error(err),
					zap.Any("event", ev))
			}
		default:
			if err := doOnceCb(); err != nil {
				router.done <- err
				return
			}
		}
	}
}

func (router *Router) Done() <-chan error {
	return router.done
}

func (router *Router) PrintStatistics() {
	router.logger.Info("router statistics",
		zap.Duration("run_time", router.runTime),
		zap.Uint64("post_count", router.postCount),
		zap.Uint64("post_fails", router.postFails),
		zap.Uint64("dispatch_count", router.dispatchCount),
		zap.Uint64("dispatch_fails", router.dispatchFails),
		zap.Float64("throughput", float64(router.postCount)/router.runTime.Seconds()))
}

func (router *Router) dispatch(ev event) error {
	switch ev.id {
	case TickEvent:
		tick, ok := ev.data.(common.Tick)
		if !ok {
			return errors.New("invalid type assertion for tick event")
		}
		if router.TickHandler != nil {
			router.TickHandler(tick)
		} else {
			router.logger.Debug("tick handler is nil")
		}
	case BarEvent:
		bar, ok := ev.data.(common.Bar)
		if !ok {
			return errors.New("invalid type assertion for bar event")
		}
		if router.BarHandler != nil {
			router.BarHandler(bar)
		} else {
			router.logger.Debug("bar handler is nil")
		}
	case EquityEvent:
		eq, ok := ev.data.(fixed.Point)
		if !ok {
			return errors.New("invalid type assertion for equity event")
		}
		if router.EquityHandler != nil {
			router.EquityHandler(eq)
		} else {
			router.logger.Debug("equity handler is nil")
		}
	case BalanceEvent:
		bal, ok := ev.data.(fixed.Point)
		if !ok {
			return errors.New("invalid type assertion for balance event")
		}
		if router.BalanceHandler != nil {
			router.BalanceHandler(bal)
		} else {
			router.logger.Debug("balance handler is nil")
		}
	case PositionOpenedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position opened event")
		}
		if router.PositionOpenedHandler != nil {
			router.PositionOpenedHandler(pos)
		} else {
			router.logger.Debug("position opened handler is nil")
		}
	case PositionClosedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position closed event")
		}
		if router.PositionClosedHandler != nil {
			router.PositionClosedHandler(pos)
		} else {
			router.logger.Debug("position closed handler is nil")
		}
	case PositionPnLUpdatedEvent:
		pos, ok := ev.data.(common.Position)
		if !ok {
			return errors.New("invalid type assertion for position pnl updated event")
		}
		if router.PositionPnLUpdatedHandler != nil {
			router.PositionPnLUpdatedHandler(pos)
		} else {
			router.logger.Debug("position pnl updated handler is nil")
		}
	case OrderEvent:
		order, ok := ev.data.(common.Order)
		if !ok {
			return errors.New("invalid type assertion for order event")
		}
		if router.OrderHandler != nil {
			router.OrderHandler(order)
		} else {
			router.logger.Debug("order handler is nil")
		}
	case SignalEvent:
		sig, ok := ev.data.(common.Signal)
		if !ok {
			return errors.New("invalid type assertion for signal event")
		}
		if router.SignalHandler != nil {
			router.SignalHandler(sig)
		} else {
			router.logger.Debug("signal handler is nil")
		}
	default:
		return fmt.Errorf("unsupported event id: %v", ev.id)
	}
	return nil
}
