package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
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
