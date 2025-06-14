package model

import (
	"go.uber.org/zap/zapcore"
	"peter-kozarec/equinox/pkg/utility/fixed"
	"time"
)

type State int
type PositionId int64

func (pId PositionId) Int64() int64 {
	return int64(pId)
}

const (
	Opened State = iota
	PendingOpen
	Closed
	PendingClose
)

type Position struct {
	Id          PositionId
	State       State
	GrossProfit fixed.Point
	NetProfit   fixed.Point
	OpenPrice   fixed.Point
	ClosePrice  fixed.Point
	OpenTime    time.Time
	CloseTime   time.Time
	Size        fixed.Point
	StopLoss    fixed.Point
	TakeProfit  fixed.Point
}

func (position *Position) IsLong() bool {
	return position.Size.Gt(fixed.Zero)
}

func (position *Position) IsShort() bool {
	return position.Size.Lt(fixed.Zero)
}

func (position *Position) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddUint64("id", uint64(position.Id))
	enc.AddInt("state", int(position.State))
	enc.AddString("gross_profit", position.GrossProfit.String())
	enc.AddString("net_profit", position.NetProfit.String())
	enc.AddString("open_price", position.OpenPrice.String())
	enc.AddString("close_price", position.ClosePrice.String())
	enc.AddString("open_time", position.OpenTime.String())
	enc.AddString("close_time", position.CloseTime.String())
	enc.AddString("size", position.Size.String())
	enc.AddString("stop_loss", position.StopLoss.String())
	enc.AddString("take_profit", position.TakeProfit.String())
	return nil
}
