package position

import (
	"errors"

	"github.com/peter-kozarec/equinox/pkg/common"
)

var (
	ErrPosNotFound = errors.New("position is not found")
)

type Holdings struct {
	positions []common.Position
}

func NewHoldings() *Holdings {
	return &Holdings{}
}

func (h *Holdings) OnPositionOpen(p common.Position) {
	h.positions = append(h.positions, p)
}

func (h *Holdings) OnPositionClose(p common.Position) {
	for idx, position := range h.positions {
		if position.Id == p.Id {
			h.positions = append(h.positions[:idx], h.positions[idx+1:]...)
			break
		}
	}
}

func (h *Holdings) OnPositionUpdate(p common.Position) {
	for idx := range h.positions {
		position := &h.positions[idx]
		if position.Id == p.Id {
			*position = p
			break
		}
	}
}

func (h *Holdings) Count() int {
	return len(h.positions)
}

func (h *Holdings) Find(id common.PositionId) (common.Position, error) {
	for _, position := range h.positions {
		if position.Id == id {
			return position, nil
		}
	}
	return common.Position{}, ErrPosNotFound
}
