package bus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"
)

func TestBusRouter_Post(t *testing.T) {
	r := NewRouter(10)

	err := r.Post(TickEvent, common.Tick{})
	if err != nil {
		t.Errorf("Post failed: %v", err)
	}

	if r.postCount.Load() != 1 {
		t.Errorf("Expected postCount=1, got %d", r.postCount.Load())
	}
}

func TestBusRouter_PostCapacityReached(t *testing.T) {
	r := NewRouter(1)

	err := r.Post(TickEvent, common.Tick{})
	if err != nil {
		t.Errorf("First Post failed: %v", err)
	}

	err = r.Post(TickEvent, common.Tick{})
	if err == nil {
		t.Error("Expected error when capacity reached")
	}

	if r.postFails.Load() != 1 {
		t.Errorf("Expected postFails=1, got %d", r.postFails.Load())
	}
}

func TestBusRouter_Exec(t *testing.T) {
	r := NewRouter(10)

	var tickHandled bool
	r.TickHandler = func(ctx context.Context, tick common.Tick) {
		tickHandled = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errChan := r.Exec(ctx)

	if err := r.Post(TickEvent, common.Tick{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	cancel()

	err := <-errChan
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	if !tickHandled {
		t.Error("Tick handler not called")
	}

	if r.dispatchCount.Load() != 1 {
		t.Errorf("Expected dispatchCount=1, got %d", r.dispatchCount.Load())
	}
}

func TestBusRouter_ExecLoop(t *testing.T) {
	r := NewRouter(10)

	var barHandled bool
	r.BarHandler = func(ctx context.Context, bar common.Bar) {
		barHandled = true
	}

	doOnceCount := 0
	doOnceCb := func() error {
		doOnceCount++
		if doOnceCount > 5 {
			return errors.New("done")
		}
		return nil
	}

	if err := r.Post(BarEvent, common.Bar{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}

	ctx := context.Background()
	errChan := r.ExecLoop(ctx, doOnceCb)

	err := <-errChan
	if err == nil || err.Error() != "done" {
		t.Errorf("Expected 'done' error, got %v", err)
	}

	if !barHandled {
		t.Error("Bar handler not called")
	}

	if doOnceCount <= 5 {
		t.Errorf("Expected doOnceCount>5, got %d", doOnceCount)
	}
}

func TestBusRouter_AllEventTypes(t *testing.T) {
	r := NewRouter(20)

	handlers := map[EventId]bool{
		TickEvent:               false,
		BarEvent:                false,
		EquityEvent:             false,
		BalanceEvent:            false,
		PositionOpenedEvent:     false,
		PositionClosedEvent:     false,
		PositionPnLUpdatedEvent: false,
		OrderEvent:              false,
		OrderAcceptedEvent:      false,
		OrderRejectedEvent:      false,
		SignalEvent:             false,
	}

	r.TickHandler = func(ctx context.Context, tick common.Tick) {
		handlers[TickEvent] = true
	}
	r.BarHandler = func(ctx context.Context, bar common.Bar) {
		handlers[BarEvent] = true
	}
	r.EquityHandler = func(ctx context.Context, eq common.Equity) {
		handlers[EquityEvent] = true
	}
	r.BalanceHandler = func(ctx context.Context, bal common.Balance) {
		handlers[BalanceEvent] = true
	}
	r.PositionOpenedHandler = func(ctx context.Context, pos common.Position) {
		handlers[PositionOpenedEvent] = true
	}
	r.PositionClosedHandler = func(ctx context.Context, pos common.Position) {
		handlers[PositionClosedEvent] = true
	}
	r.PositionPnLUpdatedHandler = func(ctx context.Context, pos common.Position) {
		handlers[PositionPnLUpdatedEvent] = true
	}
	r.OrderHandler = func(ctx context.Context, order common.Order) {
		handlers[OrderEvent] = true
	}
	r.OrderAcceptedHandler = func(ctx context.Context, oa common.OrderAccepted) {
		handlers[OrderAcceptedEvent] = true
	}
	r.OrderRejectedHandler = func(ctx context.Context, or common.OrderRejected) {
		handlers[OrderRejectedEvent] = true
	}
	r.SignalHandler = func(ctx context.Context, sig common.Signal) {
		handlers[SignalEvent] = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	errChan := r.Exec(ctx)

	if err := r.Post(TickEvent, common.Tick{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(BarEvent, common.Bar{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(EquityEvent, common.Equity{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(BalanceEvent, common.Balance{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(PositionOpenedEvent, common.Position{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(PositionClosedEvent, common.Position{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(PositionPnLUpdatedEvent, common.Position{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(OrderEvent, common.Order{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(OrderAcceptedEvent, common.OrderAccepted{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(OrderRejectedEvent, common.OrderRejected{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(SignalEvent, common.Signal{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-errChan

	for eventId, handled := range handlers {
		if !handled {
			t.Errorf("Event %d handler not called", eventId)
		}
	}

	if r.dispatchCount.Load() != 11 {
		t.Errorf("Expected dispatchCount=11, got %d", r.dispatchCount.Load())
	}
}

func TestBusRouter_InvalidTypeAssertion(t *testing.T) {
	r := NewRouter(10)

	r.TickHandler = func(ctx context.Context, tick common.Tick) {
		t.Error("Handler should not be called")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errChan := r.Exec(ctx)

	if err := r.Post(TickEvent, "invalid data type"); err != nil {
		t.Errorf("Post failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	cancel()
	<-errChan

	if r.dispatchFails.Load() != 1 {
		t.Errorf("Expected dispatchFails=1, got %d", r.dispatchFails.Load())
	}
}

func TestBusRouter_NilHandlers(t *testing.T) {
	r := NewRouter(10)

	ctx, cancel := context.WithCancel(context.Background())
	errChan := r.Exec(ctx)

	if err := r.Post(TickEvent, common.Tick{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}
	if err := r.Post(BarEvent, common.Bar{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	cancel()
	<-errChan

	if r.dispatchCount.Load() != 2 {
		t.Errorf("Expected dispatchCount=2, got %d", r.dispatchCount.Load())
	}

	if r.dispatchFails.Load() != 0 {
		t.Errorf("Expected dispatchFails=0, got %d", r.dispatchFails.Load())
	}
}

func TestBusRouter_UnsupportedEventId(t *testing.T) {
	r := NewRouter(10)

	ctx, cancel := context.WithCancel(context.Background())
	errChan := r.Exec(ctx)

	if err := r.Post(EventId(99), struct{}{}); err != nil {
		t.Errorf("Post failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	cancel()
	<-errChan

	if r.dispatchFails.Load() != 1 {
		t.Errorf("Expected dispatchFails=1, got %d", r.dispatchFails.Load())
	}
}

func TestBusRouter_ConcurrentPost(t *testing.T) {
	r := NewRouter(1000)

	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				if err := r.Post(TickEvent, common.Tick{}); err != nil {
					t.Errorf("Post failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	expectedPosts := uint64(numGoroutines * eventsPerGoroutine)
	if r.postCount.Load() != expectedPosts {
		t.Errorf("Expected postCount=%d, got %d", expectedPosts, r.postCount.Load())
	}
}

func TestBusRouter_ContextCancellation(t *testing.T) {
	r := NewRouter(10)

	ctx, cancel := context.WithCancel(context.Background())
	errChan := r.Exec(ctx)

	cancel()

	err := <-errChan
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func BenchmarkBusRouter_Post(b *testing.B) {
	r := NewRouter(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := r.Post(TickEvent, common.Tick{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
	}
}

func BenchmarkBusRouter_ConcurrentPost(b *testing.B) {
	r := NewRouter(b.N)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := r.Post(TickEvent, common.Tick{}); err != nil {
				b.Errorf("Post failed: %v", err)
			}
		}
	})
}

func BenchmarkBusRouter_AllEventTypes(b *testing.B) {
	r := NewRouter(b.N * 11)

	r.TickHandler = func(ctx context.Context, tick common.Tick) {}
	r.BarHandler = func(ctx context.Context, bar common.Bar) {}
	r.EquityHandler = func(ctx context.Context, eq common.Equity) {}
	r.BalanceHandler = func(ctx context.Context, bal common.Balance) {}
	r.PositionOpenedHandler = func(ctx context.Context, pos common.Position) {}
	r.PositionClosedHandler = func(ctx context.Context, pos common.Position) {}
	r.PositionPnLUpdatedHandler = func(ctx context.Context, pos common.Position) {}
	r.OrderHandler = func(ctx context.Context, order common.Order) {}
	r.OrderAcceptedHandler = func(ctx context.Context, oa common.OrderAccepted) {}
	r.OrderRejectedHandler = func(ctx context.Context, or common.OrderRejected) {}
	r.SignalHandler = func(ctx context.Context, sig common.Signal) {}

	ctx, cancel := context.WithCancel(context.Background())
	errChan := r.Exec(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := r.Post(TickEvent, common.Tick{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(BarEvent, common.Bar{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(EquityEvent, common.Equity{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(BalanceEvent, common.Balance{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(PositionOpenedEvent, common.Position{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(PositionClosedEvent, common.Position{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(PositionPnLUpdatedEvent, common.Position{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(OrderEvent, common.Order{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(OrderAcceptedEvent, common.OrderAccepted{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(OrderRejectedEvent, common.OrderRejected{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
		if err := r.Post(SignalEvent, common.Signal{}); err != nil {
			b.Errorf("Post failed: %v", err)
		}
	}

	cancel()
	<-errChan
}

func BenchmarkBusRouter_ExecLoop(b *testing.B) {
	r := NewRouter(1000)

	r.TickHandler = func(ctx context.Context, tick common.Tick) {}

	callCount := 0
	doOnceCb := func() error {
		callCount++
		if callCount >= b.N {
			return errors.New("done")
		}
		if callCount%100 == 0 {
			if err := r.Post(TickEvent, common.Tick{}); err != nil {
				return err
			}
		}
		return nil
	}

	ctx := context.Background()

	b.ResetTimer()
	errChan := r.ExecLoop(ctx, doOnceCb)
	<-errChan
}
