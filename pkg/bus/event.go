package bus

import (
	"github.com/peter-kozarec/equinox/pkg/common"
)

type EventId uint8

const (
	TickEvent EventId = iota
	BarEvent
	EquityEvent
	BalanceEvent
	PositionOpenedEvent
	PositionClosedEvent
	PositionPnLUpdatedEvent
	OrderEvent
	OrderRejectedEvent
	OrderAcceptedEvent
	SignalEvent
)

type TickEventHandler func(common.Tick)
type BarEventHandler func(common.Bar)
type EquityEventHandler func(common.Equity)
type BalanceEventHandler func(common.Balance)
type PositionOpenedEventHandler func(common.Position)
type PositionClosedEventHandler func(common.Position)
type PositionPnLUpdatedEventHandler func(common.Position)
type OrderEventHandler func(common.Order)
type OrderRejectedEventHandler func(common.OrderRejected)
type OrderAcceptedEventHandler func(common.OrderAccepted)
type SignalEventHandler func(common.Signal)
