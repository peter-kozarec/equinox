package cfg

import (
	"peter-kozarec/equinox/internal/utility"
	"time"
)

var LotValue = utility.MustNewFixed(100000, 0)
var PipSize = utility.MustNewFixed(1, 4)
var CommissionPerLot = utility.MustNewFixed(3, 0)
var PipSlippage = utility.MustNewFixed(10, 5)
var BarPeriod = time.Minute
var StartBalance = utility.TenThousandFixed
