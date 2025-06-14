package bus

import (
	"github.com/peter-kozarec/equinox/pkg/model"
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

type TickEventHandler func(*model.Tick)
type BarEventHandler func(*model.Bar)
type EquityEventHandler func(*fixed.Point)
type BalanceEventHandler func(*fixed.Point)
type PositionOpenedEventHandler func(*model.Position)
type PositionClosedEventHandler func(*model.Position)
type PositionPnLUpdatedEventHandler func(*model.Position)
type OrderEventHandler func(*model.Order)
