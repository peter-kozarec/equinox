package bus

import "peter-kozarec/equinox/internal/model"

type EventId uint8

const (
	TickEvent EventId = iota
	BarEvent
	EquityEvent
	BalanceEvent
)

type TickEventHandler func(*model.Tick) error
type BarEventHandler func(*model.Bar) error
type EquityEventHandler func(*model.Equity) error
type BalanceEventHandler func(*model.Balance) error
