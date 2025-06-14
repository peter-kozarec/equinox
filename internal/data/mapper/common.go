package mapper

import (
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility/fixed"
)

type BinaryTick struct {
	TimeStamp int64
	Bid       float64
	Ask       float64
	BidVolume float64
	AskVolume float64
}

func (binaryTick BinaryTick) ToModelTick(tick *model.Tick) {
	tick.TimeStamp = binaryTick.TimeStamp
	tick.Ask = fixed.FromFloat(binaryTick.Ask)
	tick.Bid = fixed.FromFloat(binaryTick.Bid)
	tick.AskVolume = fixed.FromFloat(binaryTick.AskVolume)
	tick.BidVolume = fixed.FromFloat(binaryTick.BidVolume)
}
