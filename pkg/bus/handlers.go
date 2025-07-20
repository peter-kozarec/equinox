package bus

import (
	"context"

	"github.com/peter-kozarec/equinox/pkg/common"
)

type EventHandler[T any] = func(context.Context, T)

type TickEventHandler EventHandler[common.Tick]
type BarEventHandler EventHandler[common.Bar]
type EquityEventHandler EventHandler[common.Equity]
type BalanceEventHandler EventHandler[common.Balance]
type PositionOpenEventHandler EventHandler[common.Position]
type PositionCloseEventHandler EventHandler[common.Position]
type PositionUpdateEventHandler EventHandler[common.Position]
type OrderEventHandler EventHandler[common.Order]
type OrderRejectionEventHandler EventHandler[common.OrderRejected]
type OrderAcceptanceHandler EventHandler[common.OrderAccepted]
type SignalEventHandler EventHandler[common.Signal]
type SignalRejectionEventHandler EventHandler[common.SignalRejected]
type SignalAcceptanceEventHandler EventHandler[common.SignalAccepted]

func MergeHandlers[T any](handlers ...EventHandler[T]) EventHandler[T] {
	return func(ctx context.Context, event T) {
		for _, handler := range handlers {
			handler(ctx, event)
		}
	}
}
