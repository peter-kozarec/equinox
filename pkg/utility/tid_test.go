package utility

import (
	"sync"
	"testing"
	"time"
)

func TestUtility_CreateTraceID(t *testing.T) {
	id1 := CreateTraceID()
	id2 := CreateTraceID()

	if id1 == id2 {
		t.Error("Expected different TraceIDs")
	}

	ts1, m1, s1 := ParseTraceID(id1)
	ts2, m2, s2 := ParseTraceID(id2)

	if m1 != m2 {
		t.Errorf("Machine IDs differ: %d vs %d", m1, m2)
	}

	if ts2.Before(ts1) {
		t.Error("Timestamps not monotonic")
	}

	if ts1.Equal(ts2) && s2 <= s1 {
		t.Errorf("Sequence not increasing: %d -> %d", s1, s2)
	}
}

func TestUtility_CreateTraceIDUniqueness(t *testing.T) {
	const n = 100000
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
			t.Errorf("Duplicate TraceID: %d", id)
		}
		seen[id] = true
	}
}

func TestUtility_ParseTraceID(t *testing.T) {
	before := time.Now().Truncate(time.Millisecond)
	id := CreateTraceID()
	after := time.Now().Truncate(time.Millisecond).Add(time.Millisecond)

	ts, machine, seq := ParseTraceID(id)

	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp out of range: %v not in [%v, %v]", ts, before, after)
	}

	if machine > maxMachine {
		t.Errorf("Machine ID out of range: %d > %d", machine, maxMachine)
	}

	if seq > maxSequence {
		t.Errorf("Sequence out of range: %d > %d", seq, maxSequence)
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

func BenchmarkUtility_ParseTraceID(b *testing.B) {
	id := CreateTraceID()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = ParseTraceID(id)
	}
}
