package indicators

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Atr struct {
	windowSize int

	lastClose  fixed.Point
	lastAtr    fixed.Point
	currentAtr fixed.Point
	currentTr  fixed.Point
}

func NewAtr(windowSize int) *Atr {
	return &Atr{
		windowSize: windowSize,

		lastClose:  fixed.Zero,
		lastAtr:    fixed.Zero,
		currentAtr: fixed.Zero,
		currentTr:  fixed.Zero,
	}
}

func (a *Atr) OnBar(b common.Bar) {
	defer func() {
		a.lastClose = b.Close
	}()

	if a.lastClose.IsZero() {
		return
	}

	_1 := b.High.Sub(b.Low).Abs()
	_2 := b.High.Sub(a.lastClose).Abs()
	_3 := b.Low.Sub(a.lastClose).Abs()

	a.currentTr = _1
	if _2.Gt(a.currentTr) {
		a.currentTr = _2
	}
	if _3.Gt(a.currentTr) {
		a.currentTr = _3
	}

	if a.lastAtr.IsZero() {
		a.currentAtr = a.currentTr
	} else {
		a.currentAtr = a.lastAtr.MulInt(a.windowSize - 1).Add(a.currentTr).DivInt(a.windowSize)
	}

	a.lastAtr = a.currentAtr
}

func (a *Atr) AverageTrueRange() fixed.Point {
	return a.currentAtr
}

func (a *Atr) TrueRange() fixed.Point {
	return a.currentTr
}

func (a *Atr) Ready() bool {
	return !a.lastAtr.IsZero()
}

func (a *Atr) Reset() {
	a.lastClose = fixed.Zero
	a.lastAtr = fixed.Zero
	a.currentAtr = fixed.Zero
	a.currentTr = fixed.Zero
}
