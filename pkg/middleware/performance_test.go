package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"
)

func TestMiddlewarePerformance_NewPerformance(t *testing.T) {
	p := NewPerformance()
	if p == nil {
		t.Error("NewPerformance returned nil")
		return
	}
	if p.tickEventCounter != 0 {
		t.Errorf("Expected tickEventCounter=0, got %d", p.tickEventCounter)
	}
}

func TestMiddlewarePerformance_WithTick(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, tick common.Tick) {
		handlerCalled = true
		time.Sleep(10 * time.Millisecond)
	}

	wrapped := p.WithTick(handler)
	wrapped(context.Background(), common.Tick{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.tickEventCounter != 1 {
		t.Errorf("Expected tickEventCounter=1, got %d", p.tickEventCounter)
	}

	if p.totalTickHandlerDur < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", p.totalTickHandlerDur)
	}
}

func TestMiddlewarePerformance_WithBar(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, bar common.Bar) {
		handlerCalled = true
		time.Sleep(5 * time.Millisecond)
	}

	wrapped := p.WithBar(handler)
	wrapped(context.Background(), common.Bar{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.barEventCounter != 1 {
		t.Errorf("Expected barEventCounter=1, got %d", p.barEventCounter)
	}

	if p.totalBarHandlerDur < 5*time.Millisecond {
		t.Errorf("Expected duration >= 5ms, got %v", p.totalBarHandlerDur)
	}
}

func TestMiddlewarePerformance_WithBalance(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, balance common.Balance) {
		handlerCalled = true
	}

	wrapped := p.WithBalance(handler)
	wrapped(context.Background(), common.Balance{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.balanceEventCounter != 1 {
		t.Errorf("Expected balanceEventCounter=1, got %d", p.balanceEventCounter)
	}
}

func TestMiddlewarePerformance_WithEquity(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, equity common.Equity) {
		handlerCalled = true
	}

	wrapped := p.WithEquity(handler)
	wrapped(context.Background(), common.Equity{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.equityEventCounter != 1 {
		t.Errorf("Expected equityEventCounter=1, got %d", p.equityEventCounter)
	}
}

func TestMiddlewarePerformance_WithPositionOpened(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, position common.Position) {
		handlerCalled = true
	}

	wrapped := p.WithPositionOpened(handler)
	wrapped(context.Background(), common.Position{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.positionOpenedEventCounter != 1 {
		t.Errorf("Expected positionOpenedEventCounter=1, got %d", p.positionOpenedEventCounter)
	}
}

func TestMiddlewarePerformance_WithPositionClosed(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, position common.Position) {
		handlerCalled = true
	}

	wrapped := p.WithPositionClosed(handler)
	wrapped(context.Background(), common.Position{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.positionClosedEventCounter != 1 {
		t.Errorf("Expected positionClosedEventCounter=1, got %d", p.positionClosedEventCounter)
	}
}

func TestMiddlewarePerformance_WithPositionPnLUpdated(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, position common.Position) {
		handlerCalled = true
	}

	wrapped := p.WithPositionPnLUpdated(handler)
	wrapped(context.Background(), common.Position{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.positionPnLUpdatedEventCounter != 1 {
		t.Errorf("Expected positionPnLUpdatedEventCounter=1, got %d", p.positionPnLUpdatedEventCounter)
	}
}

func TestMiddlewarePerformance_WithOrder(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, order common.Order) {
		handlerCalled = true
	}

	wrapped := p.WithOrder(handler)
	wrapped(context.Background(), common.Order{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.orderEventCounter != 1 {
		t.Errorf("Expected orderEventCounter=1, got %d", p.orderEventCounter)
	}
}

func TestMiddlewarePerformance_WithOrderRejected(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, rejected common.OrderRejected) {
		handlerCalled = true
	}

	wrapped := p.WithOrderRejected(handler)
	wrapped(context.Background(), common.OrderRejected{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.orderRejectedEventCounter != 1 {
		t.Errorf("Expected orderRejectedEventCounter=1, got %d", p.orderRejectedEventCounter)
	}
}

func TestMiddlewarePerformance_WithOrderAccepted(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, accepted common.OrderAccepted) {
		handlerCalled = true
	}

	wrapped := p.WithOrderAccepted(handler)
	wrapped(context.Background(), common.OrderAccepted{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.orderAcceptedEventCounter != 1 {
		t.Errorf("Expected orderAcceptedEventCounter=1, got %d", p.orderAcceptedEventCounter)
	}
}

func TestMiddlewarePerformance_WithSignal(t *testing.T) {
	p := NewPerformance()

	var handlerCalled bool
	handler := func(ctx context.Context, signal common.Signal) {
		handlerCalled = true
		time.Sleep(15 * time.Millisecond)
	}

	wrapped := p.WithSignal(handler)
	wrapped(context.Background(), common.Signal{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if p.signalEventCounter != 1 {
		t.Errorf("Expected signalEventCounter=1, got %d", p.signalEventCounter)
	}

	if p.totalSignalHandlerDur < 15*time.Millisecond {
		t.Errorf("Expected duration >= 15ms, got %v", p.totalSignalHandlerDur)
	}
}

func TestMiddlewarePerformance_MultipleCallsSameHandler(t *testing.T) {
	p := NewPerformance()

	callCount := 0
	handler := func(ctx context.Context, tick common.Tick) {
		callCount++
		time.Sleep(1 * time.Millisecond)
	}

	wrapped := p.WithTick(handler)

	for i := 0; i < 10; i++ {
		wrapped(context.Background(), common.Tick{})
	}

	if callCount != 10 {
		t.Errorf("Expected handler called 10 times, got %d", callCount)
	}

	if p.tickEventCounter != 10 {
		t.Errorf("Expected tickEventCounter=10, got %d", p.tickEventCounter)
	}

	if p.totalTickHandlerDur < 10*time.Millisecond {
		t.Errorf("Expected total duration >= 10ms, got %v", p.totalTickHandlerDur)
	}
}

func TestMiddlewarePerformance_AllHandlers(t *testing.T) {
	p := NewPerformance()

	handlers := map[string]bool{
		"tick":           false,
		"bar":            false,
		"balance":        false,
		"equity":         false,
		"positionOpened": false,
		"positionClosed": false,
		"positionPnL":    false,
		"order":          false,
		"orderRejected":  false,
		"orderAccepted":  false,
		"signal":         false,
	}

	p.WithTick(func(ctx context.Context, tick common.Tick) {
		handlers["tick"] = true
	})(context.Background(), common.Tick{})

	p.WithBar(func(ctx context.Context, bar common.Bar) {
		handlers["bar"] = true
	})(context.Background(), common.Bar{})

	p.WithBalance(func(ctx context.Context, balance common.Balance) {
		handlers["balance"] = true
	})(context.Background(), common.Balance{})

	p.WithEquity(func(ctx context.Context, equity common.Equity) {
		handlers["equity"] = true
	})(context.Background(), common.Equity{})

	p.WithPositionOpened(func(ctx context.Context, position common.Position) {
		handlers["positionOpened"] = true
	})(context.Background(), common.Position{})

	p.WithPositionClosed(func(ctx context.Context, position common.Position) {
		handlers["positionClosed"] = true
	})(context.Background(), common.Position{})

	p.WithPositionPnLUpdated(func(ctx context.Context, position common.Position) {
		handlers["positionPnL"] = true
	})(context.Background(), common.Position{})

	p.WithOrder(func(ctx context.Context, order common.Order) {
		handlers["order"] = true
	})(context.Background(), common.Order{})

	p.WithOrderRejected(func(ctx context.Context, rejected common.OrderRejected) {
		handlers["orderRejected"] = true
	})(context.Background(), common.OrderRejected{})

	p.WithOrderAccepted(func(ctx context.Context, accepted common.OrderAccepted) {
		handlers["orderAccepted"] = true
	})(context.Background(), common.OrderAccepted{})

	p.WithSignal(func(ctx context.Context, signal common.Signal) {
		handlers["signal"] = true
	})(context.Background(), common.Signal{})

	for name, called := range handlers {
		if !called {
			t.Errorf("Handler %s not called", name)
		}
	}

	totalEvents := p.tickEventCounter + p.barEventCounter + p.balanceEventCounter +
		p.equityEventCounter + p.positionOpenedEventCounter + p.positionClosedEventCounter +
		p.positionPnLUpdatedEventCounter + p.orderEventCounter + p.orderRejectedEventCounter +
		p.orderAcceptedEventCounter + p.signalEventCounter

	if totalEvents != 11 {
		t.Errorf("Expected total events=11, got %d", totalEvents)
	}
}

func TestMiddlewarePerformance_PrintStatistics(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	p := NewPerformance()

	p.WithTick(func(ctx context.Context, tick common.Tick) {
		time.Sleep(2 * time.Millisecond)
	})(context.Background(), common.Tick{})

	p.WithTick(func(ctx context.Context, tick common.Tick) {
		time.Sleep(3 * time.Millisecond)
	})(context.Background(), common.Tick{})

	p.WithBar(func(ctx context.Context, bar common.Bar) {
		time.Sleep(1 * time.Millisecond)
	})(context.Background(), common.Bar{})

	p.PrintStatistics()

	logs := buf.String()
	if !strings.Contains(logs, "performance statistics") {
		t.Error("Expected log message not found")
	}

	if !strings.Contains(logs, "tick_event_count=2") {
		t.Error("Expected tick_event_count=2 in logs")
	}

	if !strings.Contains(logs, "bar_event_count=1") {
		t.Error("Expected bar_event_count=1 in logs")
	}
}

func TestMiddlewarePerformance_PrintStatisticsEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	p := NewPerformance()
	p.PrintStatistics()

	logs := buf.String()
	if logs != "" {
		t.Error("Unexpected message found")
	}
}

func TestMiddlewarePerformance_ContextPropagation(t *testing.T) {
	p := NewPerformance()

	type contextKey string
	const testKey contextKey = "test"

	ctx := context.WithValue(context.Background(), testKey, "value")
	var receivedCtx context.Context

	handler := func(c context.Context, tick common.Tick) {
		receivedCtx = c
	}

	wrapped := p.WithTick(handler)
	wrapped(ctx, common.Tick{})

	if receivedCtx.Value(testKey) != "value" {
		t.Error("Context not properly propagated")
	}
}

func TestMiddlewarePerformance_ConcurrentAccess(t *testing.T) {
	p := NewPerformance()

	var wg sync.WaitGroup
	iterations := 100

	tickHandler := p.WithTick(func(ctx context.Context, tick common.Tick) {
		time.Sleep(100 * time.Microsecond)
	})

	barHandler := p.WithBar(func(ctx context.Context, bar common.Bar) {
		time.Sleep(100 * time.Microsecond)
	})

	signalHandler := p.WithSignal(func(ctx context.Context, signal common.Signal) {
		time.Sleep(100 * time.Microsecond)
	})

	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			tickHandler(context.Background(), common.Tick{})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			barHandler(context.Background(), common.Bar{})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			signalHandler(context.Background(), common.Signal{})
		}
	}()

	wg.Wait()

	if p.tickEventCounter != int64(iterations) {
		t.Errorf("Expected tickEventCounter=%d, got %d", iterations, p.tickEventCounter)
	}

	if p.barEventCounter != int64(iterations) {
		t.Errorf("Expected barEventCounter=%d, got %d", iterations, p.barEventCounter)
	}

	if p.signalEventCounter != int64(iterations) {
		t.Errorf("Expected signalEventCounter=%d, got %d", iterations, p.signalEventCounter)
	}
}

func TestMiddlewarePerformance_ZeroDuration(t *testing.T) {
	p := NewPerformance()

	handler := func(ctx context.Context, tick common.Tick) {}

	wrapped := p.WithTick(handler)
	wrapped(context.Background(), common.Tick{})

	if p.tickEventCounter != 1 {
		t.Errorf("Expected tickEventCounter=1, got %d", p.tickEventCounter)
	}

	if p.totalTickHandlerDur < 0 {
		t.Error("Duration should not be negative")
	}
}

func BenchmarkMiddlewarePerformance_WithTick(b *testing.B) {
	p := NewPerformance()
	handler := func(ctx context.Context, tick common.Tick) {}
	wrapped := p.WithTick(handler)
	ctx := context.Background()
	tick := common.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped(ctx, tick)
	}
}

func BenchmarkMiddlewarePerformance_WithAllHandlers(b *testing.B) {
	p := NewPerformance()

	tickHandler := p.WithTick(func(ctx context.Context, tick common.Tick) {})
	barHandler := p.WithBar(func(ctx context.Context, bar common.Bar) {})
	signalHandler := p.WithSignal(func(ctx context.Context, signal common.Signal) {})

	ctx := context.Background()
	tick := common.Tick{}
	bar := common.Bar{}
	signal := common.Signal{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tickHandler(ctx, tick)
		barHandler(ctx, bar)
		signalHandler(ctx, signal)
	}
}

func BenchmarkMiddlewarePerformance_ConcurrentHandlers(b *testing.B) {
	p := NewPerformance()

	handler := p.WithTick(func(ctx context.Context, tick common.Tick) {})
	ctx := context.Background()
	tick := common.Tick{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler(ctx, tick)
		}
	})
}

func BenchmarkMiddlewarePerformance_PrintStatistics(b *testing.B) {
	p := NewPerformance()

	handler := p.WithTick(func(ctx context.Context, tick common.Tick) {})
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		handler(ctx, common.Tick{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.PrintStatistics()
	}
}
