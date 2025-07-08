package common

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

func (t Tick) Average() fixed.Point {
	return t.Ask.Add(t.Bid).DivInt64(2)
}

func (t Tick) AggregatedVolume() fixed.Point {
	return t.AskVolume.Add(t.BidVolume)
}
