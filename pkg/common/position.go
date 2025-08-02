package common

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility"
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
	Id                     PositionId     `json:"id"`
	Status                 PositionStatus `json:"status"`
	Side                   PositionSide   `json:"side"`
	Size                   fixed.Point    `json:"size"`
	Margin                 fixed.Point    `json:"margin"`
	GrossProfit            fixed.Point    `json:"gross_profit"`
	NetProfit              fixed.Point    `json:"net_profit"`
	OpenPrice              fixed.Point    `json:"open_price"`
	ClosePrice             fixed.Point    `json:"close_price"`
	OpenTime               time.Time      `json:"open_time"`
	CloseTime              time.Time      `json:"close_time"`
	StopLoss               fixed.Point    `json:"stop_loss"`
	TakeProfit             fixed.Point    `json:"take_profit"`
	Commissions            fixed.Point    `json:"commission"`
	Swaps                  fixed.Point    `json:"swaps"`
	Currency               string         `json:"currency"`
	OpenExchangeRate       fixed.Point    `json:"open_exchange_rate"`
	OpenConversionFeeRate  fixed.Point    `json:"open_conversion_fee_rate"`
	OpenConversionFee      fixed.Point    `json:"open_conversion_fee"`
	CloseExchangeRate      fixed.Point    `json:"close_exchange_rate"`
	CloseConversionFeeRate fixed.Point    `json:"close_conversion_fee_rate"`
	CloseConversionFee     fixed.Point    `json:"close_conversion_fee"`
	Slippage               fixed.Point    `json:"slippage"`

	Source        string              `json:"src,omitempty"`
	Symbol        string              `json:"symbol,omitempty"`
	ExecutionID   utility.ExecutionID `json:"eid,omitempty"`
	TraceID       utility.TraceID     `json:"tid,omitempty"`
	OrderTraceIDs []utility.TraceID   `json:"order_tid,omitempty"`
	TimeStamp     time.Time           `json:"ts"`
}
