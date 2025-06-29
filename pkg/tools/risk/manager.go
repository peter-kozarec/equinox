package risk

import (
	"errors"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

var (
	ErrPosNotRecommended = errors.New("position is not recommended to open")
)

type Manager struct {
	logger *zap.Logger

	minPosSize fixed.Point
	maxPosSize fixed.Point
	sizeScale  int

	riskPercentage         fixed.Point
	volatilityFactor       bool
	drawdownFactor         bool
	signalConfidenceFactor bool

	balance fixed.Point
	equity  fixed.Point

	//avgAtr fixed.Point
	//curAtr fixed.Point
}

func NewManager(logger *zap.Logger, minPosSize, maxPosSize fixed.Point, sizeScale int, options ...ManagerOption) *Manager {
	m := &Manager{
		logger: logger,

		minPosSize: minPosSize,
		maxPosSize: maxPosSize,
		sizeScale:  sizeScale,

		riskPercentage:         fixed.Zero,
		volatilityFactor:       false,
		drawdownFactor:         false,
		signalConfidenceFactor: false,
	}

	for _, option := range options {
		option(m)
	}

	return m
}

func (m *Manager) PositionSize(_ fixed.Point) (fixed.Point, error) {
	return fixed.Zero, ErrPosNotRecommended
}

func (m *Manager) OnBar(_ common.Bar) {

}

func (m *Manager) OnBalance(balance fixed.Point) {
	m.balance = balance
}

func (m *Manager) OnEquity(equity fixed.Point) {
	m.equity = equity
}

//func clamp(base, min, max fixed.Point) fixed.Point {
//	if base.Lt(min) {
//		return min
//	} else if base.Gt(max) {
//		return max
//	}
//	return base
//}
