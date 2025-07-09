package utility

import (
	"sync"
	"sync/atomic"
	"time"
)

type TraceID = uint64

const (
	delta TraceID = 1
)

var (
	tid = atomic.Uint64{}
)

func CreateTraceID() TraceID {
	sync.OnceFunc(func() {
		tid.Store(I64ToU64Unsafe(time.Now().UnixMilli()))
	})
	return tid.Add(delta)
}
