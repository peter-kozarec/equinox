package bus

import (
	"fmt"
	"log/slog"
	"time"
)

type Statistics struct {
	RunTime       time.Duration
	PostCount     uint64
	PostFails     uint64
	DispatchCount uint64
	DispatchFails uint64
	Throughput    float64
}

func (s Statistics) Print() {
	slog.Info("router statistics",
		"run_time", fmt.Sprintf("%.2fs", s.RunTime.Seconds()),
		"post_count", s.PostCount,
		"post_fails", s.PostFails,
		"dispatch_count", s.DispatchCount,
		"dispatch_fails", s.DispatchFails,
		"throughput", fmt.Sprintf("%.2f", s.Throughput))
}
