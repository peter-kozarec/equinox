package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       fixed.Point
	Bid       fixed.Point
	AskVolume fixed.Point
	BidVolume fixed.Point
}

func (t Tick) Average() fixed.Point {
	return t.Ask.Add(t.Bid).DivInt(2)
}

func (t Tick) AggregatedVolume() fixed.Point {
	return t.AskVolume.Add(t.BidVolume)
}

func (t Tick) Fields() []zap.Field {
	return []zap.Field{
		zap.Int64("timestamp", t.TimeStamp),
		zap.String("ask", t.Ask.String()),
		zap.String("bid", t.Bid.String()),
		zap.String("ask_volume", t.AskVolume.String()),
		zap.String("bid_volume", t.BidVolume.String()),
	}
}
