package model

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type Bar struct {
	Period    time.Duration
	TimeStamp int64
	Open      fixed.Point
	High      fixed.Point
	Low       fixed.Point
	Close     fixed.Point
	Volume    fixed.Point
}

func (b Bar) Fields() []zap.Field {
	return []zap.Field{
		zap.String("period", b.Period.String()),
		zap.Int64("timestamp", b.TimeStamp),
		zap.String("open", b.Open.String()),
		zap.String("high", b.High.String()),
		zap.String("low", b.Low.String()),
		zap.String("close", b.Close.String()),
		zap.String("volume", b.Volume.String()),
	}
}
