package atr

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Indicator struct {
	windowSize int

	lastClose  fixed.Point
	lastAtr    fixed.Point
	currentAtr fixed.Point
	currentTr  fixed.Point
}

func NewIndicator(windowSize int) *Indicator {
	return &Indicator{
		windowSize: windowSize,

		lastClose:  fixed.Zero,
		lastAtr:    fixed.Zero,
		currentAtr: fixed.Zero,
		currentTr:  fixed.Zero,
	}
}

func (i *Indicator) OnBar(bar common.Bar) {
	defer func() {
		i.lastClose = bar.Close
	}()

	if i.lastClose.IsZero() {
		return
	}

	a := bar.High.Sub(bar.Low).Abs()
	b := bar.High.Sub(i.lastClose).Abs()
	c := bar.Low.Sub(i.lastClose).Abs()

	i.currentTr = a
	if b.Gt(i.currentTr) {
		i.currentTr = b
	}
	if c.Gt(i.currentTr) {
		i.currentTr = c
	}

	if i.lastAtr.IsZero() {
		i.currentAtr = i.currentTr
	} else {
		i.currentAtr = i.lastAtr.MulInt(i.windowSize - 1).Add(i.currentTr).DivInt(i.windowSize)
	}

	i.lastAtr = i.currentAtr
}

func (i *Indicator) AverageTrueRange() fixed.Point {
	return i.currentAtr
}

func (i *Indicator) TrueRange() fixed.Point {
	return i.currentTr
}

func (i *Indicator) Ready() bool {
	return !i.lastAtr.IsZero()
}

func (i *Indicator) Reset() {
	i.lastClose = fixed.Zero
	i.lastAtr = fixed.Zero
	i.currentAtr = fixed.Zero
	i.currentTr = fixed.Zero
}
