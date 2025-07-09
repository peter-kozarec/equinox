package utility

import (
	"sync"
	"testing"
)

func TestUtility_GetExecutionID(t *testing.T) {
	id1 := GetExecutionID()
	id2 := GetExecutionID()

	if id1 != id2 {
		t.Errorf("Expected same ExecutionID, got id1=%s, id2=%s", id1, id2)
	}
}

func TestUtility_GetExecutionIDConsistency(t *testing.T) {
	id := GetExecutionID()

	for i := 0; i < 1000; i++ {
		if GetExecutionID() != id {
			t.Errorf("ExecutionID changed unexpectedly")
		}
	}
}

func TestUtility_GetExecutionIDConcurrent(t *testing.T) {
	const goroutines = 100
	const callsPerGoroutine = 1000

	ids := make(chan ExecutionID, goroutines*callsPerGoroutine)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				ids <- GetExecutionID()
			}
		}()
	}

	wg.Wait()
	close(ids)

	first := <-ids
	for id := range ids {
		if id != first {
			t.Errorf("Expected all ExecutionIDs to be %s, got %s", first, id)
		}
	}
}

func TestUtility_GetExecutionIDValid(t *testing.T) {
	id := GetExecutionID()

	if id == (ExecutionID{}) {
		t.Error("ExecutionID is zero value")
	}
}

func BenchmarkUtility_GetExecutionID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetExecutionID()
	}
}

func BenchmarkUtility_GetExecutionIDParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = GetExecutionID()
		}
	})
}
