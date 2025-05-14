package bus

import "peter-kozarec/equinox/internal/model"

type EventId uint8

const (
	TickEvent EventId = iota
	BarEvent
)

type TickEventHandler func(*model.Tick) error
type BarEventHandler func(*model.Bar) error
