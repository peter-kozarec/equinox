package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/peter-kozarec/equinox/pkg/common"
)

func setupTestLogger(_ *testing.T) *bytes.Buffer {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	return buf
}

func TestMiddlewareMonitor_NewMonitor(t *testing.T) {
	m := NewMonitor(MonitorTicks | MonitorBars)
	if m.flags != (MonitorTicks | MonitorBars) {
		t.Errorf("Expected flags %d, got %d", MonitorTicks|MonitorBars, m.flags)
	}
}

func TestMiddlewareMonitor_WithTick(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, tick common.Tick) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorTicks)
	wrapped := m.WithTick(handler)

	wrapped(context.Background(), common.Tick{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "tick") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithTickNoMonitor(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, tick common.Tick) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorNone)
	wrapped := m.WithTick(handler)

	wrapped(context.Background(), common.Tick{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if strings.Contains(buf.String(), "tick") {
		t.Error("Unexpected log entry")
	}
}

func TestMiddlewareMonitor_WithTickMonitorAll(t *testing.T) {
	buf := setupTestLogger(t)

	handler := func(ctx context.Context, tick common.Tick) {}

	m := NewMonitor(MonitorAll)
	wrapped := m.WithTick(handler)

	wrapped(context.Background(), common.Tick{})

	if !strings.Contains(buf.String(), "tick") {
		t.Error("Log entry not found with MonitorAll")
	}
}

func TestMiddlewareMonitor_WithBar(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, bar common.Bar) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorBars)
	wrapped := m.WithBar(handler)

	wrapped(context.Background(), common.Bar{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "bar") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithEquity(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, equity common.Equity) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorEquity)
	wrapped := m.WithEquity(handler)

	wrapped(context.Background(), common.Equity{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "equity") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithBalance(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, balance common.Balance) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorBalance)
	wrapped := m.WithBalance(handler)

	wrapped(context.Background(), common.Balance{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "balance") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithPositionOpened(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, position common.Position) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorPositionsOpened)
	wrapped := m.WithPositionOpened(handler)

	wrapped(context.Background(), common.Position{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "position_open") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithPositionClosed(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, position common.Position) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorPositionsClosed)
	wrapped := m.WithPositionClosed(handler)

	wrapped(context.Background(), common.Position{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "position_closed") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithPositionPnLUpdated(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, position common.Position) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorPositionsPnLUpdated)
	wrapped := m.WithPositionPnLUpdated(handler)

	wrapped(context.Background(), common.Position{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "position_update") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithOrder(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, order common.Order) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorOrders)
	wrapped := m.WithOrder(handler)

	wrapped(context.Background(), common.Order{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "order") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithOrderRejected(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, rejected common.OrderRejected) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorOrdersRejected)
	wrapped := m.WithOrderRejected(handler)

	wrapped(context.Background(), common.OrderRejected{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "order_rejected") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithOrderAccepted(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, accepted common.OrderAccepted) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorOrdersAccepted)
	wrapped := m.WithOrderAccepted(handler)

	wrapped(context.Background(), common.OrderAccepted{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "order_accepted") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_WithSignal(t *testing.T) {
	buf := setupTestLogger(t)

	var handlerCalled bool
	handler := func(ctx context.Context, signal common.Signal) {
		handlerCalled = true
	}

	m := NewMonitor(MonitorSignals)
	wrapped := m.WithSignal(handler)

	wrapped(context.Background(), common.Signal{})

	if !handlerCalled {
		t.Error("Handler not called")
	}

	if !strings.Contains(buf.String(), "signal") {
		t.Error("Log entry not found")
	}
}

func TestMiddlewareMonitor_MultipleFlags(t *testing.T) {
	buf := setupTestLogger(t)

	m := NewMonitor(MonitorTicks | MonitorBars | MonitorSignals)

	tickHandler := m.WithTick(func(ctx context.Context, tick common.Tick) {})
	barHandler := m.WithBar(func(ctx context.Context, bar common.Bar) {})
	signalHandler := m.WithSignal(func(ctx context.Context, signal common.Signal) {})
	equityHandler := m.WithEquity(func(ctx context.Context, equity common.Equity) {})

	ctx := context.Background()

	buf.Reset()
	tickHandler(ctx, common.Tick{})
	if !strings.Contains(buf.String(), "tick") {
		t.Error("Tick log not found")
	}

	buf.Reset()
	barHandler(ctx, common.Bar{})
	if !strings.Contains(buf.String(), "bar") {
		t.Error("Bar log not found")
	}

	buf.Reset()
	signalHandler(ctx, common.Signal{})
	if !strings.Contains(buf.String(), "signal") {
		t.Error("Signal log not found")
	}

	buf.Reset()
	equityHandler(ctx, common.Equity{})
	if strings.Contains(buf.String(), "equity") {
		t.Error("Unexpected equity log")
	}
}

func TestMiddlewareMonitor_MonitorAllOverride(t *testing.T) {
	buf := setupTestLogger(t)

	m := NewMonitor(MonitorAll)

	handlers := []struct {
		name    string
		execute func()
	}{
		{
			"tick",
			func() {
				h := m.WithTick(func(ctx context.Context, tick common.Tick) {})
				h(context.Background(), common.Tick{})
			},
		},
		{
			"bar",
			func() {
				h := m.WithBar(func(ctx context.Context, bar common.Bar) {})
				h(context.Background(), common.Bar{})
			},
		},
		{
			"equity",
			func() {
				h := m.WithEquity(func(ctx context.Context, equity common.Equity) {})
				h(context.Background(), common.Equity{})
			},
		},
		{
			"balance",
			func() {
				h := m.WithBalance(func(ctx context.Context, balance common.Balance) {})
				h(context.Background(), common.Balance{})
			},
		},
		{
			"position_open",
			func() {
				h := m.WithPositionOpened(func(ctx context.Context, position common.Position) {})
				h(context.Background(), common.Position{})
			},
		},
		{
			"position_closed",
			func() {
				h := m.WithPositionClosed(func(ctx context.Context, position common.Position) {})
				h(context.Background(), common.Position{})
			},
		},
		{
			"position_update",
			func() {
				h := m.WithPositionPnLUpdated(func(ctx context.Context, position common.Position) {})
				h(context.Background(), common.Position{})
			},
		},
		{
			"order",
			func() {
				h := m.WithOrder(func(ctx context.Context, order common.Order) {})
				h(context.Background(), common.Order{})
			},
		},
		{
			"order_rejected",
			func() {
				h := m.WithOrderRejected(func(ctx context.Context, rejected common.OrderRejected) {})
				h(context.Background(), common.OrderRejected{})
			},
		},
		{
			"order_accepted",
			func() {
				h := m.WithOrderAccepted(func(ctx context.Context, accepted common.OrderAccepted) {})
				h(context.Background(), common.OrderAccepted{})
			},
		},
		{
			"signal",
			func() {
				h := m.WithSignal(func(ctx context.Context, signal common.Signal) {})
				h(context.Background(), common.Signal{})
			},
		},
	}

	for _, h := range handlers {
		buf.Reset()
		h.execute()
		if !strings.Contains(buf.String(), h.name) {
			t.Errorf("Expected log for %s with MonitorAll", h.name)
		}
	}
}

func TestMiddlewareMonitor_ContextPropagation(t *testing.T) {
	m := NewMonitor(MonitorNone)

	type contextKey string
	const testKey contextKey = "test"

	ctx := context.WithValue(context.Background(), testKey, "value")
	var receivedCtx context.Context

	handler := func(c context.Context, tick common.Tick) {
		receivedCtx = c
	}

	wrapped := m.WithTick(handler)
	wrapped(ctx, common.Tick{})

	if receivedCtx.Value(testKey) != "value" {
		t.Error("Context not properly propagated")
	}
}

func TestMiddlewareMonitor_FlagCombinations(t *testing.T) {
	tests := []struct {
		name     string
		flags    MonitorFlags
		expected []string
	}{
		{
			name:     "None",
			flags:    MonitorNone,
			expected: []string{},
		},
		{
			name:     "Single flag",
			flags:    MonitorTicks,
			expected: []string{"tick"},
		},
		{
			name:     "Multiple flags",
			flags:    MonitorTicks | MonitorBars | MonitorOrders,
			expected: []string{"tick", "bar", "order"},
		},
		{
			name:     "All flags",
			flags:    MonitorAll,
			expected: []string{"tick", "bar", "equity", "balance", "position_open", "position_closed", "position_update", "order", "order_rejected", "order_accepted", "signal"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := setupTestLogger(t)
			m := NewMonitor(test.flags)
			ctx := context.Background()

			m.WithTick(func(ctx context.Context, tick common.Tick) {})(ctx, common.Tick{})
			m.WithBar(func(ctx context.Context, bar common.Bar) {})(ctx, common.Bar{})
			m.WithEquity(func(ctx context.Context, equity common.Equity) {})(ctx, common.Equity{})
			m.WithBalance(func(ctx context.Context, balance common.Balance) {})(ctx, common.Balance{})
			m.WithPositionOpened(func(ctx context.Context, position common.Position) {})(ctx, common.Position{})
			m.WithPositionClosed(func(ctx context.Context, position common.Position) {})(ctx, common.Position{})
			m.WithPositionPnLUpdated(func(ctx context.Context, position common.Position) {})(ctx, common.Position{})
			m.WithOrder(func(ctx context.Context, order common.Order) {})(ctx, common.Order{})
			m.WithOrderRejected(func(ctx context.Context, rejected common.OrderRejected) {})(ctx, common.OrderRejected{})
			m.WithOrderAccepted(func(ctx context.Context, accepted common.OrderAccepted) {})(ctx, common.OrderAccepted{})
			m.WithSignal(func(ctx context.Context, signal common.Signal) {})(ctx, common.Signal{})

			logs := buf.String()
			for _, expected := range test.expected {
				if !strings.Contains(logs, expected) {
					t.Errorf("Expected log containing '%s' not found", expected)
				}
			}
		})
	}
}

func BenchmarkMiddlewareMonitor_WithTickEnabled(b *testing.B) {
	m := NewMonitor(MonitorTicks)
	handler := func(ctx context.Context, tick common.Tick) {}
	wrapped := m.WithTick(handler)
	ctx := context.Background()
	tick := common.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped(ctx, tick)
	}
}

func BenchmarkMiddlewareMonitor_WithTickDisabled(b *testing.B) {
	m := NewMonitor(MonitorNone)
	handler := func(ctx context.Context, tick common.Tick) {}
	wrapped := m.WithTick(handler)
	ctx := context.Background()
	tick := common.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped(ctx, tick)
	}
}

func BenchmarkMiddlewareMonitor_WithAllEnabled(b *testing.B) {
	m := NewMonitor(MonitorAll)
	handler := func(ctx context.Context, tick common.Tick) {}
	wrapped := m.WithTick(handler)
	ctx := context.Background()
	tick := common.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped(ctx, tick)
	}
}

func BenchmarkMiddlewareMonitor_MultipleHandlers(b *testing.B) {
	m := NewMonitor(MonitorAll)

	tickHandler := m.WithTick(func(ctx context.Context, tick common.Tick) {})
	barHandler := m.WithBar(func(ctx context.Context, bar common.Bar) {})
	orderHandler := m.WithOrder(func(ctx context.Context, order common.Order) {})

	ctx := context.Background()
	tick := common.Tick{}
	bar := common.Bar{}
	order := common.Order{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tickHandler(ctx, tick)
		barHandler(ctx, bar)
		orderHandler(ctx, order)
	}
}
