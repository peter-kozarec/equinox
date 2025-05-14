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
	tickHandler TickEventHandler

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
		logger:        logger,
		done:          make(chan error),
		events:        make(chan event, eventCapacity),
		tickHandler:   nil,
		runTime:       0,
		postCount:     0,
		postFails:     0,
		dispatchCount: 0,
		dispatchFails: 0,
		loopCycles:    0,
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

func (router *Router) Subscribe(id EventId, handler interface{}) error {
	switch id {
	case TickEvent:
		tickHandler, ok := handler.(TickEventHandler)
		if !ok {
			return errors.New("invalid type assertion for tick event handler")
		}
		router.tickHandler = tickHandler
	default:
		return errors.New(fmt.Sprintf("unsupported event id: %v", id))
	}
	return nil
}

func (router *Router) Run(ctx context.Context, executorLoop func(context.Context) error) {

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

func (router *Router) dispatch(ev event) error {
	switch ev.id {
	case TickEvent:
		tick, ok := ev.data.(*model.Tick)
		if !ok {
			return errors.New("invalid type assertion for tick event")
		}
		return router.tickHandler(tick)
	default:
		return errors.New(fmt.Sprintf("unsupported event id: %v", ev.id))
	}
}
