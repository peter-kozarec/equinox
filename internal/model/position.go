package model

import (
	"github.com/govalues/decimal"
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
	Id          PositionId // Unique identifier
	State       State
	GrossProfit decimal.Decimal // PnL before costs
	NetProfit   decimal.Decimal // PnL after slippage, commission, swaps
	PnL         Price           // PnL in micro pips
	OpenPrice   Price
	ClosePrice  Price
	OpenTime    time.Time
	CloseTime   time.Time
	Size        Size // Positive = long, Negative = short
	StopLoss    Price
	TakeProfit  Price
}
