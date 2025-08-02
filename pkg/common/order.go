package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"time"
)

type OrderCommand int
type OrderType int
type OrderSide int
type TimeInForce int

const (
	OrderCommandPositionOpen OrderCommand = iota
	OrderCommandPositionClose
	OrderCommandPositionModify
)

const (
	OrderTypeMarket OrderType = iota
	OrderTypeLimit
)

const (
	OrderSideBuy OrderSide = iota
	OrderSideSell
)

const (
	TimeInForceGoodTillCancel TimeInForce = iota
	TimeInForceImmediateOrCancel
	TimeInForceFillOrKill
	TimeInForceGoodTillDate
)

type Order struct {
	Command     OrderCommand `json:"command"`
	Type        OrderType    `json:"type"`
	Side        OrderSide    `json:"side"`
	Price       fixed.Point  `json:"price"`
	Size        fixed.Point  `json:"size"`
	FilledSize  fixed.Point  `json:"filled_size"`
	TimeInForce TimeInForce  `json:"time_in_force"`
	ExpireTime  time.Time    `json:"expire_time"`
	StopLoss    fixed.Point  `json:"stop_loss,omitempty"`
	TakeProfit  fixed.Point  `json:"take_profit,omitempty"`
	PositionId  PositionId   `json:"position_id,omitempty"`
	Comment     string       `json:"comment,omitempty"`

	Source      string              `json:"src,omitempty"`
	Symbol      string              `json:"symbol,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}

type OrderRejected struct {
	OriginalOrder Order  `json:"original_order"`
	Reason        string `json:"reason,omitempty"`

	Source      string              `json:"src,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}

type OrderAccepted struct {
	OriginalOrder Order `json:"original_order"`

	Source      string              `json:"src,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}

type OrderFilled struct {
	OriginalOrder Order      `json:"original_order"`
	PositionId    PositionId `json:"position_id"`

	Source      string              `json:"src,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}

type OrderCancelled struct {
	OriginalOrder Order       `json:"original_order"`
	CancelledSize fixed.Point `json:"cancelled_size"`

	Source      string              `json:"src,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}
