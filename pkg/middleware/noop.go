package middleware

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

//goland:noinspection ALL
var (
	NoopTickHdl    = func(common.Tick) {}
	NoopBarHdl     = func(common.Bar) {}
	NoopEquityHdl  = func(fixed.Point) {}
	NoopBalanceHdl = func(fixed.Point) {}
	NoopPosOpnHdl  = func(common.Position) {}
	NoopPosUpdHdl  = func(common.Position) {}
	NoopPosClsHdl  = func(common.Position) {}
	NoopOrderHdl   = func(common.Order) {}
	NoopSignalHdl  = func(common.Signal) {}
)
