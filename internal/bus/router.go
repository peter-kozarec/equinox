package bus

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility"
	"time"
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

	// Statistics
	runTime       time.Duration
	postCount     uint64
	postFails     uint64
	dispatchCount uint64
	dispatchFails uint64
	cycles        uint64
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

func (router *Router) Exec(ctx context.Context, cycle time.Duration) {

	router.runTime = 0
	router.dispatchCount = 0
	router.dispatchFails = 0
	router.postCount = 0
	router.postFails = 0
	router.cycles = 0

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
			router.cycles++
			time.Sleep(cycle)
		}
	}
}

func (router *Router) ExecLoop(ctx context.Context, executorLoop func(context.Context) error) {

	router.runTime = 0
	router.dispatchCount = 0
	router.dispatchFails = 0
	router.postCount = 0
	router.postFails = 0
	router.cycles = 0

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
			router.cycles++
			if err := executorLoop(ctx); err != nil {
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
		zap.Uint64("throughput", router.postCount/uint64(router.runTime.Seconds())),
		zap.Uint64("cycles", router.cycles))
}

func (router *Router) dispatch(ev event) error {
	switch ev.id {
	case TickEvent:
		tick, ok := ev.data.(*model.Tick)
		if !ok {
			return errors.New("invalid type assertion for tick event")
		}
		return router.TickHandler(tick)
	case BarEvent:
		bar, ok := ev.data.(*model.Bar)
		if !ok {
			return errors.New("invalid type assertion for bar event")
		}
		return router.BarHandler(bar)
	case EquityEvent:
		eq, ok := ev.data.(*utility.Fixed)
		if !ok {
			return errors.New("invalid type assertion for equity event")
		}
		return router.EquityHandler(eq)
	case BalanceEvent:
		bal, ok := ev.data.(*utility.Fixed)
		if !ok {
			return errors.New("invalid type assertion for balance event")
		}
		return router.BalanceHandler(bal)
	case PositionOpenedEvent:
		pos, ok := ev.data.(*model.Position)
		if !ok {
			return errors.New("invalid type assertion for position opened event")
		}
		return router.PositionOpenedHandler(pos)
	case PositionClosedEvent:
		pos, ok := ev.data.(*model.Position)
		if !ok {
			return errors.New("invalid type assertion for position closed event")
		}
		return router.PositionClosedHandler(pos)
	case PositionPnLUpdatedEvent:
		pos, ok := ev.data.(*model.Position)
		if !ok {
			return errors.New("invalid type assertion for position pnl updated event")
		}
		return router.PositionPnLUpdatedHandler(pos)
	case OrderEvent:
		order, ok := ev.data.(*model.Order)
		if !ok {
			return errors.New("invalid type assertion for order event")
		}
		return router.OrderHandler(order)
	default:
		return errors.New(fmt.Sprintf("unsupported event id: %v", ev.id))
	}
}
