package bus

import "peter-kozarec/equinox/internal/model"

type EventId uint8

const (
	TickEvent EventId = iota
)

type TickEventHandler func(*model.Tick) error
