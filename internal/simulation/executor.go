package simulation

import (
	"context"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/data/mapper"
	"peter-kozarec/equinox/internal/model"
	"time"
)

type Executor struct {
	logger    *zap.Logger
	simulator *Simulator
	reader    *mapper.Reader[model.Tick]

	from int64
	to   int64
	idx  int64

	tick    model.Tick
	lastErr error
}

func NewExecutor(logger *zap.Logger, simulator *Simulator, reader *mapper.Reader[model.Tick], from, to time.Time) *Executor {
	return &Executor{
		logger:    logger,
		simulator: simulator,
		reader:    reader,
		from:      from.UnixNano(),
		to:        to.UnixNano(),
	}
}

func (e *Executor) Feed(_ context.Context) error {

	// Read the next tick from the reader
	if e.lastErr = e.reader.Read(e.idx, &e.tick); e.lastErr != nil {
		return e.lastErr
	}
	e.idx++

	// Skip ticks outside the time range
	if e.tick.TimeStamp < e.from {
		return nil
	}

	if e.tick.TimeStamp > e.to {
		return mapper.EOF
	}

	// Feed ticks to simulation
	if e.lastErr = e.simulator.OnTick(&e.tick); e.lastErr != nil {
		return e.lastErr
	}

	return nil
}
