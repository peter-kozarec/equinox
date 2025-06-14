package model

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       fixed.Point
	Bid       fixed.Point
	AskVolume fixed.Point
	BidVolume fixed.Point
}

func (tick Tick) Average() fixed.Point {
	return tick.Ask.Add(tick.Bid).DivInt(2)
}

func (tick Tick) Volume() fixed.Point {
	return tick.AskVolume.Add(tick.BidVolume)
}
