package bus

import (
	"context"
	"github.com/peter-kozarec/equinox/pkg/common"
)

type TickEventHandler func(context.Context, common.Tick)
type BarEventHandler func(context.Context, common.Bar)
type EquityEventHandler func(context.Context, common.Equity)
type BalanceEventHandler func(context.Context, common.Balance)
type PositionOpenedEventHandler func(context.Context, common.Position)
type PositionClosedEventHandler func(context.Context, common.Position)
type PositionPnLUpdatedEventHandler func(context.Context, common.Position)
type OrderEventHandler func(context.Context, common.Order)
type OrderRejectedEventHandler func(context.Context, common.OrderRejected)
type OrderAcceptedEventHandler func(context.Context, common.OrderAccepted)
type SignalEventHandler func(context.Context, common.Signal)

func MergeHandlers[T any](handlers ...func(context.Context, T)) func(context.Context, T) {
	return func(ctx context.Context, event T) {
		for _, handler := range handlers {
			handler(ctx, event)
		}
	}
}
