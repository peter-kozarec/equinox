package historical

import (
	"fmt"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility"
)

const (
	invalidIndex            = -1
	tickReaderComponentName = "datasource.historical.reader"
)

type TickReader struct {
	source *Source[BinaryTick]

	symbol string
	from   int64
	to     int64
	idx    int64
}

func NewTickReader(source *Source[BinaryTick], symbol string, from, to time.Time) *TickReader {
	return &TickReader{
		source: source,
		symbol: symbol,
		from:   from.UnixNano(),
		to:     to.UnixNano(),
		idx:    invalidIndex,
	}
}

func (t *TickReader) GetNext() (common.Tick, error) {

	var tick common.Tick
	var binTick BinaryTick

	if t.idx == invalidIndex {
		if err := t.lookupStartIndex(); err != nil {
			return tick, err
		}
	}

	if err := t.source.Read(t.idx, &binTick); err != nil {
		return tick, fmt.Errorf("error reading entry at index %d: %w", t.idx, err)
	}
	t.idx++

	if binTick.TimeStamp < t.from {
		return tick, fmt.Errorf("timestamp is not from the proposed range")
	}

	if binTick.TimeStamp > t.to {
		return tick, ErrEof
	}

	binTick.ToModelTick(&tick)

	tick.Source = tickReaderComponentName
	tick.Symbol = t.symbol
	tick.ExecutionId = utility.GetExecutionID()
	tick.TraceID = utility.CreateTraceID()

	return tick, nil
}

func (t *TickReader) lookupStartIndex() error {
	entryCount, err := t.source.EntryCount()
	if err != nil {
		return fmt.Errorf("error getting entry count: %w", err)
	}

	if entryCount == 0 {
		return fmt.Errorf("entry count is zero")
	}

	var entry BinaryTick

	low := int64(0)
	high := entryCount - 1

	for low <= high {
		mid := (low + high) / 2

		if err := t.source.Read(mid, &entry); err != nil {
			return fmt.Errorf("error reading entry at index %d: %w", mid, err)
		}

		if entry.TimeStamp < t.from {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	if low >= entryCount {
		return fmt.Errorf("no entry found with timestamp >= from")
	}

	t.idx = low
	return nil
}
