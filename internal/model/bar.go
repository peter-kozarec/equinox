package model

import (
	"go.uber.org/zap/zapcore"
	"peter-kozarec/equinox/internal/utility/fixed"
	"time"
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

func (bar *Bar) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("period", bar.Period.String())
	enc.AddString("time_stamp", time.Unix(0, bar.TimeStamp).String())
	enc.AddString("open", bar.Open.String())
	enc.AddString("high", bar.High.String())
	enc.AddString("low", bar.Low.String())
	enc.AddString("close", bar.Close.String())
	enc.AddString("volume", bar.Volume.String())
	return nil
}
