package bus

import (
	"github.com/peter-kozarec/equinox/pkg/common"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
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
)

type TickEventHandler func(common.Tick)
type BarEventHandler func(common.Bar)
type EquityEventHandler func(fixed.Point)
type BalanceEventHandler func(fixed.Point)
type PositionOpenedEventHandler func(common.Position)
type PositionClosedEventHandler func(common.Position)
type PositionPnLUpdatedEventHandler func(common.Position)
type OrderEventHandler func(common.Order)
