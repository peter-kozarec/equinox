package middleware

import (
	"context"

	"github.com/peter-kozarec/equinox/pkg/common"
)

var (
	NoopTickHandler             = func(context.Context, common.Tick) {}
	NoopBarHandler              = func(context.Context, common.Bar) {}
	NoopEquityHandler           = func(context.Context, common.Equity) {}
	NoopBalanceHandler          = func(context.Context, common.Balance) {}
	NoopPositionOpenHandler     = func(context.Context, common.Position) {}
	NoopPositionCloseHandler    = func(context.Context, common.Position) {}
	NoopPositionUpdateHandler   = func(context.Context, common.Position) {}
	NoopOrderHandler            = func(context.Context, common.Order) {}
	NoopOrderRejectionHandler   = func(context.Context, common.OrderRejected) {}
	NoopOrderAcceptanceHandler  = func(context.Context, common.OrderAccepted) {}
	NoopOrderFilledHandler      = func(context.Context, common.OrderFilled) {}
	NoopOrderCancelledHandler   = func(context.Context, common.OrderCancelled) {}
	NoopSignalHandler           = func(context.Context, common.Signal) {}
	NoopSignalRejectionHandler  = func(context.Context, common.SignalRejected) {}
	NoopSignalAcceptanceHandler = func(context.Context, common.SignalAccepted) {}
)
