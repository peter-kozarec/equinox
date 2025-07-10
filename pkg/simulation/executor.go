package simulation

import (
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"github.com/peter-kozarec/equinox/pkg/utility"

	"time"
)

type Executor struct {
	router *bus.Router
	reader *mapper.Reader[mapper.BinaryTick]

	symbol string
	from   int64
	to     int64
	idx    int64

	binaryTick mapper.BinaryTick
	tick       common.Tick
	lastErr    error
}

func NewExecutor(router *bus.Router, reader *mapper.Reader[mapper.BinaryTick], symbol string, from, to time.Time) *Executor {
	return &Executor{
		router: router,
		reader: reader,
		symbol: symbol,
		from:   from.UnixNano(),
		to:     to.UnixNano(),
	}
}

func (e *Executor) LookupStartIndex() error {
	entryCount, err := e.reader.EntryCount()
	if err != nil {
		return fmt.Errorf("error getting entry count: %w", err)
	}

	if entryCount == 0 {
		return fmt.Errorf("entry count is zero")
	}

	var entry mapper.BinaryTick

	low := int64(0)
	high := entryCount - 1

	for low <= high {
		mid := (low + high) / 2

		if err := e.reader.Read(mid, &entry); err != nil {
			return fmt.Errorf("error reading entry at index %d: %w", mid, err)
		}

		if entry.TimeStamp < e.from {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	if low >= entryCount {
		return fmt.Errorf("no entry found with timestamp >= from")
	}

	e.idx = low
	return nil
}

func (e *Executor) DoOnce() error {

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
		return mapper.ErrEof
	}

	e.binaryTick.ToModelTick(&e.tick)

	e.tick.Source = componentName
	e.tick.Symbol = e.symbol
	e.tick.ExecutionId = utility.GetExecutionID()
	e.tick.TraceID = utility.CreateTraceID()

	// Feed ticks to simulation
	if e.lastErr = e.router.Post(bus.TickEvent, e.tick); e.lastErr != nil {
		return e.lastErr
	}

	return nil
}
