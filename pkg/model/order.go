package model

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
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
	Price      fixed.Point
	Size       fixed.Point
	StopLoss   fixed.Point
	TakeProfit fixed.Point
}

func (o Order) Fields() []zap.Field {
	return []zap.Field{
		zap.Int("command", int(o.Command)),
		zap.Int("order_type", int(o.OrderType)),
		zap.Int("position_id", int(o.PositionId)),
		zap.String("price", o.Price.String()),
		zap.String("size", o.Size.String()),
		zap.String("stop_loss", o.StopLoss.String()),
		zap.String("take_profit", o.TakeProfit.String()),
	}
}
