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
	sequence  atomic.Uint64
	machineID uint64
	epoch     = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
)

func init() {
	machineID = uint64(uuid.New().ID()) & maxMachine
}

func CreateTraceID() TraceID {
	timestamp := uint64(time.Now().UnixMilli() - epoch)
	seq := sequence.Add(1) & maxSequence

	if seq == 0 {
		time.Sleep(time.Millisecond)
		timestamp = uint64(time.Now().UnixMilli() - epoch)
	}

	return (timestamp << timestampShift) | (machineID << machineShift) | seq
}

func ParseTraceID(id TraceID) (timestamp time.Time, machine uint64, seq uint64) {
	seq = id & maxSequence
	machine = (id >> machineShift) & maxMachine
	ts := id >> timestampShift
	timestamp = time.UnixMilli(epoch + int64(ts))
	return
}
