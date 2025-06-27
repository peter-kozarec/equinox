package position

import (
	"errors"
	"testing"

	"github.com/peter-kozarec/equinox/pkg/common"
)

func TestHoldings_OnPositionOpen(t *testing.T) {
	h := NewHoldings()
	pos := common.Position{Id: 1}
	h.OnPositionOpen(pos)

	if count := h.Count(); count != 1 {
		t.Errorf("OnPositionOpen: expected count 1, got %d", count)
	}

	found, err := h.Find(pos.Id)
	if err != nil {
		t.Errorf("OnPositionOpen: expected to find position with id %v, got error: %v", pos.Id, err)
	}
	if found.Id != pos.Id {
		t.Errorf("OnPositionOpen: expected found position id %v, got %v", pos.Id, found.Id)
	}
}

func TestHoldings_OnPositionClose(t *testing.T) {
	h := NewHoldings()
	pos1 := common.Position{Id: 1}
	pos2 := common.Position{Id: 2}

	h.OnPositionOpen(pos1)
	h.OnPositionOpen(pos2)

	if count := h.Count(); count != 2 {
		t.Errorf("OnPositionClose: expected count 2, got %d", count)
	}

	h.OnPositionClose(pos1)
	if count := h.Count(); count != 1 {
		t.Errorf("OnPositionClose: expected count 1 after closing pos1, got %d", count)
	}

	_, err := h.Find(pos1.Id)
	if err == nil {
		t.Errorf("OnPositionClose: expected error when finding closed position with id %v", pos1.Id)
	}
}

func TestHoldings_OnPositionUpdate(t *testing.T) {
	h := NewHoldings()
	pos := common.Position{Id: 1}
	h.OnPositionOpen(pos)

	// Simulate an update. If common.Position had additional fields, update them.
	updatedPos := common.Position{Id: 1}
	h.OnPositionUpdate(updatedPos)

	found, err := h.Find(pos.Id)
	if err != nil {
		t.Errorf("OnPositionUpdate: expected to find updated position, got error: %v", err)
	}
	if found.Id != updatedPos.Id {
		t.Errorf("OnPositionUpdate: expected position id %v, got %v", updatedPos.Id, found.Id)
	}
}

func TestHoldings_FindNotFound(t *testing.T) {
	h := NewHoldings()
	_, err := h.Find(999)
	if !errors.Is(err, PositionNotFound) {
		t.Errorf("TestFindNotFound: expected PositionNotFound error, got %v", err)
	}
}

func TestHoldings_MultipleOperations(t *testing.T) {
	h := NewHoldings()
	positions := []common.Position{
		{Id: 1},
		{Id: 2},
		{Id: 3},
	}

	// Open positions
	for _, pos := range positions {
		h.OnPositionOpen(pos)
	}
	if count := h.Count(); count != len(positions) {
		t.Errorf("TestMultipleOperations: expected count %d, got %d", len(positions), count)
	}

	// Update the second position.
	updatedPos2 := common.Position{Id: 2}
	h.OnPositionUpdate(updatedPos2)

	// Close the third position.
	h.OnPositionClose(positions[2])
	if count := h.Count(); count != 2 {
		t.Errorf("TestMultipleOperations: expected count 2 after closing one position, got %d", count)
	}

	// Confirm the third position is removed.
	_, err := h.Find(positions[2].Id)
	if err == nil {
		t.Errorf("TestMultipleOperations: expected error for position id %v after closing", positions[2].Id)
	}
}

func BenchmarkHoldings_OnPositionOpen(b *testing.B) {
	h := NewHoldings()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := common.Position{Id: common.PositionId(i)}
		h.OnPositionOpen(pos)
	}
}

func BenchmarkHoldings_Find(b *testing.B) {
	h := NewHoldings()
	total := 10000
	// Prepopulate the holdings.
	for i := 0; i < total; i++ {
		h.OnPositionOpen(common.Position{Id: common.PositionId(i)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = h.Find(common.PositionId(i % total))
	}
}

func BenchmarkHoldings_OnPositionUpdate(b *testing.B) {
	h := NewHoldings()
	total := 10000
	// Prepopulate the holdings.
	for i := 0; i < total; i++ {
		h.OnPositionOpen(common.Position{Id: common.PositionId(i)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := common.PositionId(i % total)
		h.OnPositionUpdate(common.Position{Id: id})
	}
}

func BenchmarkHoldings_OnPositionClose(b *testing.B) {
	h := NewHoldings()
	total := b.N
	positions := make([]common.Position, total)
	// Prepopulate the holdings.
	for j := 0; j < total; j++ {
		pos := common.Position{Id: common.PositionId(j)}
		positions[j] = pos
		h.OnPositionOpen(pos)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Close a middle position.
		h.OnPositionClose(positions[total/2])
	}
}
