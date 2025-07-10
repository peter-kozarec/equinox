package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type PositionSide int
type PositionStatus string
type PositionId = int64

const (
	PositionSideLong PositionSide = iota
	PositionSideShort
)

const (
	PositionStatusOpen   PositionStatus = "open"
	PositionStatusClosed PositionStatus = "closed"
)

type Position struct {
	Source        string              `json:"src,omitempty"`
	Symbol        string              `json:"symbol,omitempty"`
	ExecutionID   utility.ExecutionID `json:"eid,omitempty"`
	TraceID       utility.TraceID     `json:"tid,omitempty"`
	OrderTraceIDs []utility.TraceID   `json:"order_tid,omitempty"`
	TimeStamp     time.Time           `json:"ts"`
	Id            PositionId          `json:"id"`
	Status        PositionStatus      `json:"status"`
	Side          PositionSide        `json:"side"`
	GrossProfit   fixed.Point         `json:"gross_profit"`
	NetProfit     fixed.Point         `json:"net_profit"`
	OpenPrice     fixed.Point         `json:"open_price"`
	ClosePrice    fixed.Point         `json:"close_price,omitempty"`
	OpenTime      time.Time           `json:"open_time"`
	CloseTime     time.Time           `json:"close_time,omitempty"`
	Size          fixed.Point         `json:"size"`
	StopLoss      fixed.Point         `json:"stop_loss,omitempty"`
	TakeProfit    fixed.Point         `json:"take_profit,omitempty"`
	Commission    fixed.Point         `json:"commission"`
}
