package model

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
	Price      Price
	Size       Size
	StopLoss   Price
	TakeProfit Price
}
