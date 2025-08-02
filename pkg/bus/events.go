package bus

type EventId uint8

const (
	TickEvent EventId = iota
	BarEvent
	EquityEvent
	BalanceEvent
	PositionOpenEvent
	PositionCloseEvent
	PositionUpdateEvent
	OrderEvent
	OrderRejectionEvent
	OrderAcceptanceEvent
	OrderFilledEvent
	OrderCancelledEvent
	SignalEvent
	SignalRejectionEvent
	SignalAcceptanceEvent
)
