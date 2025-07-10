package mapper

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"time"
)

type BinaryTick struct {
	TimeStamp int64
	Bid       float64
	Ask       float64
	BidVolume float64
	AskVolume float64
}

func (binaryTick BinaryTick) ToModelTick(tick *common.Tick) {
	tick.TimeStamp = time.Unix(0, binaryTick.TimeStamp)
	tick.Ask = fixed.FromFloat64(binaryTick.Ask)
	tick.Bid = fixed.FromFloat64(binaryTick.Bid)
	tick.AskVolume = fixed.FromFloat64(binaryTick.AskVolume)
	tick.BidVolume = fixed.FromFloat64(binaryTick.BidVolume)
}
