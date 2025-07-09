package utility

import (
	"sync"
	"testing"
)

func TestUtility_GetExecutionID(t *testing.T) {
	id1 := GetExecutionID()
	id2 := GetExecutionID()

	if id1 != id2 {
		t.Error("Expected same ExecutionID")
	}

	if id1.Version() != 7 {
		t.Errorf("Expected UUID v7, got v%d", id1.Version())
	}
}

func TestUtility_ResetExecutionID(t *testing.T) {
	oldID := GetExecutionID()
	newID := ResetExecutionID()

	if oldID == newID {
		t.Error("ResetExecutionID didn't change ID")
	}

	if GetExecutionID() != newID {
		t.Error("GetExecutionID doesn't return new ID")
	}
}

func TestUtility_GetExecutionIDConcurrent(t *testing.T) {
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make([]ExecutionID, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = GetExecutionID()
		}(i)
	}

	wg.Wait()

	first := results[0]
	for i, id := range results {
		if id != first {
			t.Errorf("Goroutine %d got different ID", i)
		}
	}
}

func BenchmarkUtility_GetExecutionID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetExecutionID()
	}
}
