package model

import "peter-kozarec/equinox/internal/utility"

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       utility.Fixed
	Bid       utility.Fixed
	AskVolume int32
	BidVolume int32
}

func (tick Tick) Mean() utility.Fixed {
	return tick.Ask.Add(tick.Bid).DivInt(2)
}

func (tick Tick) Volume() int32 {
	return tick.AskVolume + tick.BidVolume
}
