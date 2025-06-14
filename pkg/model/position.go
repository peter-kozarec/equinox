package model

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
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
