package model

import (
	"go.uber.org/zap/zapcore"
	"peter-kozarec/equinox/internal/utility"
	"time"
)

type Bar struct {
	Period    time.Duration
	TimeStamp int64
	Open      utility.Fixed
	High      utility.Fixed
	Low       utility.Fixed
	Close     utility.Fixed
	Volume    int32
}

func (bar *Bar) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("period", bar.Period.String())
	enc.AddInt64("time_stamp", bar.TimeStamp)
	enc.AddString("open", bar.Open.String())
	enc.AddString("high", bar.High.String())
	enc.AddString("low", bar.Low.String())
	enc.AddString("close", bar.Close.String())
	enc.AddInt32("volume", bar.Volume)
	return nil
}
