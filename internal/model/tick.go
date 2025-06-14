package model

import (
	"go.uber.org/zap/zapcore"
	"peter-kozarec/equinox/internal/utility/fixed"
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

func (tick Tick) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("time_stamp", tick.TimeStamp)
	enc.AddString("ask", tick.Ask.String())
	enc.AddString("bid", tick.Bid.String())
	enc.AddString("ask_volume", tick.AskVolume.String())
	enc.AddString("bid_volume", tick.BidVolume.String())
	return nil
}
