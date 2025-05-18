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
	reader    *mapper.Reader[mapper.BinaryTick]

	from int64
	to   int64
	idx  int64

	binaryTick mapper.BinaryTick
	tick       model.Tick
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

	e.binaryTick.ToModelTick(InstrumentPrecision, &e.tick)

	// Feed ticks to simulation
	if e.lastErr = e.simulator.OnTick(&e.tick); e.lastErr != nil {
		return e.lastErr
	}

	return nil
}
