package bus

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/model"
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
	TickHandler TickEventHandler
	BarHandler  BarEventHandler

	// Statistics
	runTime       time.Duration
	postCount     int64
	postFails     int64
	dispatchCount int64
	dispatchFails int64
	loopCycles    int64
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

func (router *Router) Exec(ctx context.Context, executorLoop func(context.Context) error) {

	router.runTime = 0
	router.dispatchCount = 0
	router.dispatchFails = 0
	router.postCount = 0
	router.postFails = 0
	router.loopCycles = 0

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
			router.loopCycles++
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
		zap.Int64("dispatch_count", router.dispatchCount),
		zap.Int64("dispatch_fails", router.dispatchFails),
		zap.Int64("post_count", router.postCount),
		zap.Int64("post_fails", router.postFails),
		zap.Int64("loop_cycles", router.loopCycles))
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
	default:
		return errors.New(fmt.Sprintf("unsupported event id: %v", ev.id))
	}
}
