package cfg

import (
	"peter-kozarec/equinox/internal/utility/fixed"
	"time"
)

var LotValue = fixed.New(10, 0)
var PipSize = fixed.New(1, 4)
var CommissionPerLot = fixed.New(3, 0)
var PipSlippage = fixed.New(10, 5)
var BarPeriod = time.Minute
var StartBalance = fixed.New(10000, 0)
