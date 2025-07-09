package utility

import (
	"sync"
	"testing"
)

func TestUtility_CreateTraceID(t *testing.T) {
	id1 := CreateTraceID()
	id2 := CreateTraceID()

	if id1 >= id2 {
		t.Errorf("Expected id2 > id1, got id1=%d, id2=%d", id1, id2)
	}

	if id2-id1 != delta {
		t.Errorf("Expected delta=%d, got %d", delta, id2-id1)
	}
}

func TestUtility_CreateTraceIDUniqueness(t *testing.T) {
	const n = 10000
	ids := make(map[TraceID]bool, n)

	for i := 0; i < n; i++ {
		id := CreateTraceID()
		if ids[id] {
			t.Errorf("Duplicate TraceID: %d", id)
		}
		ids[id] = true
	}
}

func TestUtility_CreateTraceIDConcurrent(t *testing.T) {
	const goroutines = 100
	const idsPerGoroutine = 1000

	ids := make(chan TraceID, goroutines*idsPerGoroutine)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				ids <- CreateTraceID()
			}
		}()
	}

	wg.Wait()
	close(ids)

	seen := make(map[TraceID]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("Duplicate TraceID in concurrent test: %d", id)
		}
		seen[id] = true
	}

	if len(seen) != goroutines*idsPerGoroutine {
		t.Errorf("Expected %d unique IDs, got %d", goroutines*idsPerGoroutine, len(seen))
	}
}

func TestUtility_CreateTraceIDSequence(t *testing.T) {
	start := CreateTraceID()

	for i := 1; i <= 100; i++ {
		id := CreateTraceID()
		expected := start + uint64(i)*delta
		if id != expected {
			t.Errorf("Expected ID %d, got %d", expected, id)
		}
	}
}

func BenchmarkUtility_CreateTraceID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CreateTraceID()
	}
}

func BenchmarkUtility_CreateTraceIDParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = CreateTraceID()
		}
	})
}
