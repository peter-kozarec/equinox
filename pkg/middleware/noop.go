package middleware

import (
	"github.com/peter-kozarec/equinox/pkg/model"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

//goland:noinspection ALL
var (
	NoopTickHdl    = func(model.Tick) {}
	NoopBarHdl     = func(model.Bar) {}
	NoopEquityHdl  = func(fixed.Point) {}
	NoopBalanceHdl = func(fixed.Point) {}
	NoopPosOpnHdl  = func(model.Position) {}
	NoopPosUpdHdl  = func(model.Position) {}
	NoopPosClsHdl  = func(model.Position) {}
	NoopOrderHdl   = func(model.Order) {}
)
