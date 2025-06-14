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

type TickEventHandler func(*model.Tick) error
type BarEventHandler func(*model.Bar) error
type EquityEventHandler func(*fixed.Point) error
type BalanceEventHandler func(*fixed.Point) error
type PositionOpenedEventHandler func(*model.Position) error
type PositionClosedEventHandler func(*model.Position) error
type PositionPnLUpdatedEventHandler func(*model.Position) error
type OrderEventHandler func(*model.Order) error
