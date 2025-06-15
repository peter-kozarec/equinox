package model

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
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

func (p Position) IsLong() bool {
	return p.Size.Gt(fixed.Zero)
}

func (p Position) IsShort() bool {
	return p.Size.Lt(fixed.Zero)
}

func (p Position) Fields() []zap.Field {
	return []zap.Field{
		zap.Int("id", int(p.Id)),
		zap.Int("state", int(p.State)),
		zap.String("gross_profit", p.GrossProfit.String()),
		zap.String("net_profit", p.NetProfit.String()),
		zap.String("open_price", p.OpenPrice.String()),
		zap.String("close_price", p.ClosePrice.String()),
		zap.String("open_time", p.OpenTime.String()),
		zap.String("close_time", p.CloseTime.String()),
		zap.String("size", p.Size.String()),
		zap.String("stop_loss", p.StopLoss.String()),
		zap.String("take_profit", p.TakeProfit.String()),
	}
}
