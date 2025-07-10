package middleware

import (
	"github.com/peter-kozarec/equinox/pkg/common"
)

//goland:noinspection ALL
var (
	NoopTickHdl      = func(common.Tick) {}
	NoopBarHdl       = func(common.Bar) {}
	NoopEquityHdl    = func(common.Equity) {}
	NoopBalanceHdl   = func(common.Balance) {}
	NoopPosOpnHdl    = func(common.Position) {}
	NoopPosUpdHdl    = func(common.Position) {}
	NoopPosClsHdl    = func(common.Position) {}
	NoopOrderHdl     = func(common.Order) {}
	NoopOrderRjctHdl = func(common.OrderRejected) {}
	NoopOrderAccHdl  = func(common.OrderAccepted) {}
	NoopSignalHdl    = func(common.Signal) {}
)
