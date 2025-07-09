package utility

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type TraceID = uint64

const (
	machineBits  = 10
	sequenceBits = 13

	maxSequence = 1<<sequenceBits - 1
	maxMachine  = 1<<machineBits - 1

	timestampShift = machineBits + sequenceBits
	machineShift   = sequenceBits
)

var (
	sequence      atomic.Uint64
	machineID     uint64
	epoch         = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	lastTimestamp atomic.Int64
)

func init() {
	machineID = uint64(uuid.New().ID()) & maxMachine
}

func CreateTraceID() TraceID {
	for {
		timestamp := time.Now().UnixMilli() - epoch
		last := lastTimestamp.Load()

		if timestamp < last {
			timestamp = last
		}

		if timestamp == last {
			seq := sequence.Add(1) & maxSequence
			if seq != 0 {
				return (uint64(timestamp) << timestampShift) | (machineID << machineShift) | seq
			}
			time.Sleep(time.Millisecond)
			continue
		}

		if lastTimestamp.CompareAndSwap(last, timestamp) {
			sequence.Store(0)
			seq := sequence.Add(1) & maxSequence
			return (uint64(timestamp) << timestampShift) | (machineID << machineShift) | seq
		}
	}
}

func ParseTraceID(id TraceID) (timestamp time.Time, machine uint64, seq uint64) {
	seq = id & maxSequence
	machine = (id >> machineShift) & maxMachine
	ts := id >> timestampShift
	timestamp = time.UnixMilli(epoch + int64(ts))
	return
}
