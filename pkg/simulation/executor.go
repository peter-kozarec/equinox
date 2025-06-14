package simulation

import (
	"context"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"github.com/peter-kozarec/equinox/pkg/model"
	"go.uber.org/zap"
	"time"
)

type Executor struct {
	logger    *zap.Logger
	simulator *Simulator
	reader    *mapper.Reader[mapper.BinaryTick]

	from int64
	to   int64
	idx  int64

	binaryTick mapper.BinaryTick
	lastErr    error
}

func NewExecutor(logger *zap.Logger, simulator *Simulator, reader *mapper.Reader[mapper.BinaryTick], from, to time.Time) *Executor {
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
	if e.lastErr = e.reader.Read(e.idx, &e.binaryTick); e.lastErr != nil {
		return e.lastErr
	}
	e.idx++

	// Skip ticks outside the time range
	if e.binaryTick.TimeStamp < e.from {
		return nil
	}

	if e.binaryTick.TimeStamp > e.to {
		return mapper.EOF
	}

	var tick model.Tick
	e.binaryTick.ToModelTick(&tick)

	// Feed ticks to simulation
	if e.lastErr = e.simulator.OnTick(tick); e.lastErr != nil {
		return e.lastErr
	}

	return nil
}
