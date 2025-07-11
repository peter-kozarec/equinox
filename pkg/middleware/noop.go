package middleware

import (
	"context"

	"github.com/peter-kozarec/equinox/pkg/common"
)

//goland:noinspection ALL
var (
	NoopTickHdl      = func(context.Context, common.Tick) {}
	NoopBarHdl       = func(context.Context, common.Bar) {}
	NoopEquityHdl    = func(context.Context, common.Equity) {}
	NoopBalanceHdl   = func(context.Context, common.Balance) {}
	NoopPosOpnHdl    = func(context.Context, common.Position) {}
	NoopPosUpdHdl    = func(context.Context, common.Position) {}
	NoopPosClsHdl    = func(context.Context, common.Position) {}
	NoopOrderHdl     = func(context.Context, common.Order) {}
	NoopOrderRjctHdl = func(context.Context, common.OrderRejected) {}
	NoopOrderAccHdl  = func(context.Context, common.OrderAccepted) {}
	NoopSignalHdl    = func(context.Context, common.Signal) {}
)
