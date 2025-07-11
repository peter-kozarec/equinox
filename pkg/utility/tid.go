package utility

import (
	"sync"
	"sync/atomic"
	"time"
)

type TraceID = uint64

const (
	delta TraceID = 1
	base  TraceID = 1e10
)

var (
	tid  = atomic.Uint64{}
	once = sync.Once{}
)

func CreateTraceID() TraceID {
	once.Do(func() {
		tid.Store(I64ToU64Unsafe(time.Now().Unix()) * base)
	})
	return tid.Add(delta)
}
