package model

import (
	"go.uber.org/zap/zapcore"
	"peter-kozarec/equinox/internal/utility/fixed"
)

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       fixed.Point
	Bid       fixed.Point
	AskVolume int64
	BidVolume int64
}

func (tick Tick) Average() fixed.Point {
	return tick.Ask.Add(tick.Bid).DivInt(2)
}

func (tick Tick) Volume() int64 {
	return tick.AskVolume + tick.BidVolume
}

func (tick Tick) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("time_stamp", tick.TimeStamp)
	enc.AddString("ask", tick.Ask.String())
	enc.AddString("bid", tick.Bid.String())
	enc.AddInt64("ask_volume", tick.AskVolume)
	enc.AddInt64("bid_volume", tick.BidVolume)
	return nil
}
