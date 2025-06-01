package bus

import (
	"context"
	"errors"

	"go.uber.org/zap/zaptest"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility/fixed"
	"sync"
	"testing"
	"time"
)

// Mock handlers for testing
func mockTickHandler(tick *model.Tick) error {
	return nil
}

func mockBarHandler(bar *model.Bar) error {
	return nil
}

func mockEquityHandler(equity *fixed.Point) error {
	return nil
}

func mockBalanceHandler(balance *fixed.Point) error {
	return nil
}

func mockPositionOpenedHandler(pos *model.Position) error {
	return nil
}

func mockPositionClosedHandler(pos *model.Position) error {
	return nil
}

func mockPositionPnLUpdatedHandler(pos *model.Position) error {
	return nil
}

func mockOrderHandler(order *model.Order) error {
	return nil
}

func mockFailingHandler(tick *model.Tick) error {
	return errors.New("handler failed")
}

func setupRouter(t *testing.T) *Router {
	logger := zaptest.NewLogger(t)
	router := NewRouter(logger, 10)

	// Set up all handlers
	router.TickHandler = mockTickHandler
	router.BarHandler = mockBarHandler
	router.EquityHandler = mockEquityHandler
	router.BalanceHandler = mockBalanceHandler
	router.PositionOpenedHandler = mockPositionOpenedHandler
	router.PositionClosedHandler = mockPositionClosedHandler
	router.PositionPnLUpdatedHandler = mockPositionPnLUpdatedHandler
	router.OrderHandler = mockOrderHandler

	return router
}

func Test_NewRouter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewRouter(logger, 5)

	if router.logger != logger {
		t.Error("logger not set correctly")
	}

	if cap(router.events) != 5 {
		t.Errorf("expected event capacity 5, got %d", cap(router.events))
	}

	if router.done == nil {
		t.Error("done channel not initialized")
	}
}

func Test_PostSuccess(t *testing.T) {
	router := setupRouter(t)
	tick := &model.Tick{}

	err := router.Post(TickEvent, tick)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if router.postCount != 1 {
		t.Errorf("expected post count 1, got %d", router.postCount)
	}

	if router.postFails != 0 {
		t.Errorf("expected post fails 0, got %d", router.postFails)
	}
}

func Test_PostCapacityReached(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewRouter(logger, 1) // Small capacity
	tick := &model.Tick{}

	// Fill the channel
	err := router.Post(TickEvent, tick)
	if err != nil {
		t.Errorf("first post should succeed, got %v", err)
	}

	// This should fail due to capacity
	err = router.Post(TickEvent, tick)
	if err == nil {
		t.Error("expected error when capacity reached")
	}

	if router.postFails != 1 {
		t.Errorf("expected post fails 1, got %d", router.postFails)
	}
}

func Test_DispatchAllEventTypes(t *testing.T) {
	router := setupRouter(t)

	tests := []struct {
		name    string
		eventId EventId
		data    interface{}
		wantErr bool
	}{
		{"tick event", TickEvent, &model.Tick{}, false},
		{"bar event", BarEvent, &model.Bar{}, false},
		{"equity event", EquityEvent, &fixed.Point{}, false},
		{"balance event", BalanceEvent, &fixed.Point{}, false},
		{"position opened event", PositionOpenedEvent, &model.Position{}, false},
		{"position closed event", PositionClosedEvent, &model.Position{}, false},
		{"position pnl updated event", PositionPnLUpdatedEvent, &model.Position{}, false},
		{"order event", OrderEvent, &model.Order{}, false},
		{"unsupported event", EventId(99), &model.Tick{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := event{id: tt.eventId, data: tt.data}
			err := router.dispatch(ev)

			if (err != nil) != tt.wantErr {
				t.Errorf("dispatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_DispatchTypeAssertionFailure(t *testing.T) {
	router := setupRouter(t)

	// Wrong type for tick event
	ev := event{id: TickEvent, data: &model.Bar{}}
	err := router.dispatch(ev)

	if err == nil {
		t.Error("expected error for wrong type assertion")
	}

	if err.Error() != "invalid type assertion for tick event" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func Test_ExecContextCancellation(t *testing.T) {
	router := setupRouter(t)
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		router.Exec(ctx)
	}()

	// Cancel context
	cancel()

	// Wait for done signal
	select {
	case err := <-router.Done():
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for done signal")
	}

	wg.Wait()
}

func Test_ExecEventProcessing(t *testing.T) {
	router := setupRouter(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processedEvents int
	var mu sync.Mutex

	// Override tick handler to count processed events
	router.TickHandler = func(tick *model.Tick) error {
		mu.Lock()
		processedEvents++
		mu.Unlock()
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		router.Exec(ctx)
	}()

	// Post some events
	tick := &model.Tick{}
	for i := 0; i < 3; i++ {
		err := router.Post(TickEvent, tick)
		if err != nil {
			t.Errorf("failed to post event: %v", err)
		}
	}

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for done signal or timeout
	select {
	case <-router.Done():
		// Expected - router finished
	case <-time.After(time.Second):
		t.Error("timeout waiting for router to finish")
	}

	wg.Wait()

	mu.Lock()
	if processedEvents != 3 {
		t.Errorf("expected 3 processed events, got %d", processedEvents)
	}
	mu.Unlock()

	if router.dispatchCount != 3 {
		t.Errorf("expected dispatch count 3, got %d", router.dispatchCount)
	}
}

func Test_ExecHandlerFailure(t *testing.T) {
	router := setupRouter(t)
	router.TickHandler = mockFailingHandler

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		router.Exec(ctx)
	}()

	// Post an event that will fail
	tick := &model.Tick{}
	err := router.Post(TickEvent, tick)
	if err != nil {
		t.Errorf("failed to post event: %v", err)
	}

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for done signal or timeout
	select {
	case <-router.Done():
		// Expected - router finished
	case <-time.After(time.Second):
		t.Error("timeout waiting for router to finish")
	}

	wg.Wait()

	if router.dispatchFails != 1 {
		t.Errorf("expected dispatch fails 1, got %d", router.dispatchFails)
	}
}

func Test_ExecLoopWithExecutorLoop(t *testing.T) {
	router := setupRouter(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var executorCalls int
	var mu sync.Mutex

	executorLoop := func(ctx context.Context) error {
		mu.Lock()
		executorCalls++
		mu.Unlock()

		// Add small delay to prevent tight loop
		time.Sleep(time.Millisecond)
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		router.ExecLoop(ctx, executorLoop)
	}()

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for done signal or timeout
	select {
	case <-router.Done():
		// Expected - router finished
	case <-time.After(time.Second):
		t.Error("timeout waiting for router to finish")
	}

	wg.Wait()

	mu.Lock()
	if executorCalls == 0 {
		t.Error("executor loop was never called")
	}
	mu.Unlock()
}

func Test_ExecLoopExecutorError(t *testing.T) {
	router := setupRouter(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedErr := errors.New("executor error")
	executorLoop := func(ctx context.Context) error {
		return expectedErr
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		router.ExecLoop(ctx, executorLoop)
	}()

	// Wait for done signal
	select {
	case err := <-router.Done():
		if err != expectedErr {
			t.Errorf("expected executor error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for done signal")
	}

	wg.Wait()
}

func Test_Statistics(t *testing.T) {
	router := setupRouter(t)

	// Post some events
	tick := &model.Tick{}
	router.Post(TickEvent, tick)
	router.Post(TickEvent, tick)

	// Simulate some failures
	router.postFails = 1
	router.dispatchFails = 1

	// Set some runtime
	router.runTime = time.Second

	// This should not panic and should log statistics
	router.PrintStatistics()

	// Verify statistics are tracked
	if router.postCount != 2 {
		t.Errorf("expected post count 2, got %d", router.postCount)
	}
}

func Test_ConcurrentPosting(t *testing.T) {
	router := setupRouter(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		router.Exec(ctx)
	}()

	// Start multiple goroutines posting events
	numGoroutines := 10
	eventsPerGoroutine := 10

	var postWg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		postWg.Add(1)
		go func() {
			defer postWg.Done()
			tick := &model.Tick{}
			for j := 0; j < eventsPerGoroutine; j++ {
				router.Post(TickEvent, tick)
			}
		}()
	}

	postWg.Wait()
	time.Sleep(50 * time.Millisecond) // Allow processing
	cancel()

	// Wait for done signal or timeout
	select {
	case <-router.Done():
		// Expected - router finished
	case <-time.After(time.Second):
		t.Error("timeout waiting for router to finish")
	}

	wg.Wait()

	expectedTotal := uint64(numGoroutines * eventsPerGoroutine)
	if router.postCount+router.postFails != expectedTotal {
		t.Errorf("expected total posts %d, got %d", expectedTotal, router.postCount+router.postFails)
	}
}

func Test_StatisticsReset(t *testing.T) {
	router := setupRouter(t)

	// Set some initial values
	router.postCount = 5
	router.dispatchCount = 3
	router.runTime = time.Second

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		router.Exec(ctx)
	}()

	// Cancel immediately to check reset
	cancel()

	// Wait for done signal or timeout
	select {
	case <-router.Done():
		// Expected - router finished
	case <-time.After(time.Second):
		t.Error("timeout waiting for router to finish")
	}

	wg.Wait()

	// Statistics should be reset at start of Exec
	if router.postCount != 0 || router.dispatchCount != 0 {
		t.Error("statistics should be reset at start of Exec")
	}
}

// Benchmarks
func setupBenchmarkRouter(b *testing.B) *Router {
	logger := zaptest.NewLogger(b)
	router := NewRouter(logger, 10)

	// Set up all handlers
	router.TickHandler = mockTickHandler
	router.BarHandler = mockBarHandler
	router.EquityHandler = mockEquityHandler
	router.BalanceHandler = mockBalanceHandler
	router.PositionOpenedHandler = mockPositionOpenedHandler
	router.PositionClosedHandler = mockPositionClosedHandler
	router.PositionPnLUpdatedHandler = mockPositionPnLUpdatedHandler
	router.OrderHandler = mockOrderHandler

	return router
}

func Benchmark_Post(b *testing.B) {
	router := setupBenchmarkRouter(b)
	tick := &model.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := router.Post(TickEvent, tick)
		if err != nil {
			// Channel full, drain it
			select {
			case <-router.events:
			default:
			}
			// Try again
			router.Post(TickEvent, tick)
		}
	}
}

func Benchmark_Dispatch(b *testing.B) {
	router := setupBenchmarkRouter(b)
	tick := &model.Tick{}
	ev := event{id: TickEvent, data: tick}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := router.dispatch(ev)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_DispatchAllEventTypes(b *testing.B) {
	router := setupBenchmarkRouter(b)

	events := []event{
		{TickEvent, &model.Tick{}},
		{BarEvent, &model.Bar{}},
		{EquityEvent, &fixed.Point{}},
		{BalanceEvent, &fixed.Point{}},
		{PositionOpenedEvent, &model.Position{}},
		{PositionClosedEvent, &model.Position{}},
		{PositionPnLUpdatedEvent, &model.Position{}},
		{OrderEvent, &model.Order{}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ev := events[i%len(events)]
		err := router.dispatch(ev)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_EndToEnd(b *testing.B) {
	router := setupBenchmarkRouter(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start router
	go router.Exec(ctx)

	tick := &model.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := router.Post(TickEvent, tick)
		if err != nil {
			// If channel is full, skip this iteration
			// In real scenarios, you'd handle backpressure differently
			continue
		}
	}

	// Give time for processing
	time.Sleep(time.Millisecond)
}

func Benchmark_ConcurrentPost(b *testing.B) {
	router := setupBenchmarkRouter(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start router
	go router.Exec(ctx)

	tick := &model.Tick{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := router.Post(TickEvent, tick)
			if err != nil {
				// Channel full, continue
				continue
			}
		}
	})

	// Give time for processing
	time.Sleep(time.Millisecond)
}

func Benchmark_LargeCapacity(b *testing.B) {
	logger := zaptest.NewLogger(b)
	router := NewRouter(logger, 10000) // Large capacity
	router.TickHandler = mockTickHandler

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go router.Exec(ctx)

	tick := &model.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Post(TickEvent, tick)
	}

	time.Sleep(10 * time.Millisecond)
}

func Benchmark_SmallCapacity(b *testing.B) {
	logger := zaptest.NewLogger(b)
	router := NewRouter(logger, 1) // Small capacity
	router.TickHandler = mockTickHandler

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go router.Exec(ctx)

	tick := &model.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Post(TickEvent, tick)
	}

	time.Sleep(10 * time.Millisecond)
}

func Benchmark_ExecLoop(b *testing.B) {
	router := setupBenchmarkRouter(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	execCount := 0
	executorLoop := func(ctx context.Context) error {
		execCount++
		return nil
	}

	go router.ExecLoop(ctx, executorLoop)

	tick := &model.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := router.Post(TickEvent, tick)
		if err != nil {
			continue
		}
	}

	time.Sleep(time.Millisecond)
}

// Benchmark different handler complexities
func Benchmark_FastHandler(b *testing.B) {
	logger := zaptest.NewLogger(b)
	router := NewRouter(logger, 1000)

	// Very fast handler
	router.TickHandler = func(tick *model.Tick) error {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go router.Exec(ctx)

	tick := &model.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Post(TickEvent, tick)
	}

	time.Sleep(time.Millisecond)
}

func Benchmark_SlowHandler(b *testing.B) {
	logger := zaptest.NewLogger(b)
	router := NewRouter(logger, 1000)

	// Slower handler that does some work
	router.TickHandler = func(tick *model.Tick) error {
		// Simulate some processing time
		for i := 0; i < 100; i++ {
			_ = i * i
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go router.Exec(ctx)

	tick := &model.Tick{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Post(TickEvent, tick)
	}

	time.Sleep(10 * time.Millisecond)
}

func Benchmark_MixedEventTypes(b *testing.B) {
	router := setupBenchmarkRouter(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go router.Exec(ctx)

	events := []struct {
		id   EventId
		data interface{}
	}{
		{TickEvent, &model.Tick{}},
		{BarEvent, &model.Bar{}},
		{EquityEvent, &fixed.Point{}},
		{OrderEvent, &model.Order{}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ev := events[i%len(events)]
		err := router.Post(ev.id, ev.data)
		if err != nil {
			continue
		}
	}

	time.Sleep(time.Millisecond)
}

// Memory allocation benchmarks
func Benchmark_EventAllocation(b *testing.B) {
	router := setupBenchmarkRouter(b)
	tick := &model.Tick{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// This will allocate the event struct
		router.Post(TickEvent, tick)
		// Drain to prevent blocking
		select {
		case <-router.events:
		default:
		}
	}
}

func Benchmark_Statistics(b *testing.B) {
	router := setupBenchmarkRouter(b)

	// Set up some statistics
	router.postCount = 1000
	router.dispatchCount = 950
	router.postFails = 50
	router.dispatchFails = 5
	router.runTime = time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.PrintStatistics()
	}
}
