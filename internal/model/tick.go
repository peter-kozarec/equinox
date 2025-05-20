package model

import (
	"github.com/govalues/decimal"
	"go.uber.org/zap/zapcore"
	"peter-kozarec/equinox/internal/utility"
)

var decimalTwo, _ = decimal.New(2, 0)

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       utility.Fixed
	Bid       utility.Fixed
	AskVolume int32
	BidVolume int32
}

func (tick Tick) Average() utility.Fixed {
	return tick.Ask.Add(tick.Bid).DivInt(2)
}

func (tick Tick) Volume() int32 {
	return tick.AskVolume + tick.BidVolume
}

func (tick Tick) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("time_stamp", tick.TimeStamp)
	enc.AddString("ask", tick.Ask.String())
	enc.AddString("bid", tick.Bid.String())
	enc.AddInt32("ask_volume", tick.AskVolume)
	enc.AddInt32("bid_volume", tick.BidVolume)
	return nil
}
