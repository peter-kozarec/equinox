package model

import (
	"go.uber.org/zap/zapcore"
	"peter-kozarec/equinox/internal/utility"
)

type OrderType int
type Command int

const (
	CmdOpen Command = iota
	CmdClose
	CmdModify
	CmdRemove
)

const (
	Market OrderType = iota
	Limit
)

type Order struct {
	Command    Command
	OrderType  OrderType
	PositionId PositionId
	Price      utility.Fixed
	Size       utility.Fixed
	StopLoss   utility.Fixed
	TakeProfit utility.Fixed
}

func (order *Order) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("command", int(order.Command))
	enc.AddInt("order_type", int(order.OrderType))
	enc.AddInt("position_id", int(order.PositionId))
	enc.AddString("price", order.Price.String())
	enc.AddString("size", order.Size.String())
	enc.AddString("stop_loss", order.StopLoss.String())
	enc.AddString("take_profit", order.TakeProfit.String())
	return nil
}
