package utility

import "sync/atomic"

type TraceID = uint64

const (
	delta TraceID = 1
)

var (
	tid = atomic.Uint64{}
)

func CreateTraceID() TraceID {
	return tid.Add(delta)
}
