package model

import (
	"peter-kozarec/equinox/internal/utility"
	"time"
)

type State int
type PositionId uint64

const (
	Opened State = iota
	PendingOpen
	Closed
	PendingClose
)

type Position struct {
	Id          PositionId
	State       State
	GrossProfit utility.Fixed
	NetProfit   utility.Fixed
	PipPnL      utility.Fixed
	OpenPrice   utility.Fixed
	ClosePrice  utility.Fixed
	OpenTime    time.Time
	CloseTime   time.Time
	Size        utility.Fixed
	StopLoss    utility.Fixed
	TakeProfit  utility.Fixed
}

func (position *Position) IsLong() bool {
	return position.StopLoss.Value > 0
}

func (position *Position) IsShort() bool {
	return position.StopLoss.Value < 0
}
