package bus

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"peter-kozarec/equinox/internal/model"
	"sync/atomic"
	"testing"
	"time"
)

func Test_RouterPostAndDispatch(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewRouter(logger, 10)

	var tickHandled atomic.Bool
	router.TickHandler = func(tick *model.Tick) error {
		tickHandled.Store(true)
		return nil
	}

	tick := &model.Tick{TimeStamp: time.Now().UnixNano()}
	if err := router.Post(TickEvent, tick); err != nil {
		t.Fatalf("failed to post tick event: %v", err)
	}

	exec := func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go router.ExecLoop(ctx, exec)

	select {
	case <-router.Done():
	case <-time.After(500 * time.Millisecond):
		if !tickHandled.Load() {
			t.Error("tick handler was not called")
		}
	}
}

func Test_RouterEventCapacity(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewRouter(logger, 1)

	router.TickHandler = func(tick *model.Tick) error { return nil }
	tick := &model.Tick{TimeStamp: time.Now().UnixNano()}

	_ = router.Post(TickEvent, tick)    // should succeed
	err := router.Post(TickEvent, tick) // should fail (buffer full)
	if err == nil {
		t.Error("expected error when posting beyond capacity")
	}
}

func Test_RouterDoneContextCancel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewRouter(logger, 10)

	ctx, cancel := context.WithCancel(context.Background())
	go router.ExecLoop(ctx, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	cancel()
	select {
	case err := <-router.Done():
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("router did not signal done after context cancel")
	}
}

func Benchmark_RouterPost(b *testing.B) {
	logger := zap.NewNop()
	router := NewRouter(logger, b.N)
	tick := &model.Tick{TimeStamp: time.Now().UnixNano()}
	for i := 0; i < b.N; i++ {
		_ = router.Post(TickEvent, tick)
	}
}

func Benchmark_RouterDispatch(b *testing.B) {
	logger := zap.NewNop()
	router := NewRouter(logger, b.N)
	tick := &model.Tick{TimeStamp: time.Now().UnixNano()}
	router.TickHandler = func(tick *model.Tick) error { return nil }
	for i := 0; i < b.N; i++ {
		_ = router.Post(TickEvent, tick)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go router.ExecLoop(ctx, func(ctx context.Context) error {
		time.Sleep(time.Microsecond)
		return nil
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.Post(TickEvent, tick)
	}
}
