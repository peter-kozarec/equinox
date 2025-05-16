package bus

import "peter-kozarec/equinox/internal/model"

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
type EquityEventHandler func(*model.Equity) error
type BalanceEventHandler func(*model.Balance) error
type PositionOpenedEventHandler func(*model.Position) error
type PositionClosedEventHandler func(*model.Position) error
type PositionPnLUpdatedEventHandler func(*model.Position) error
type OrderEventHandler func(*model.Order) error
