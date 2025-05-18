package model

import "peter-kozarec/equinox/internal/utility"

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
