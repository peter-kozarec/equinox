package simulation

import (
	"peter-kozarec/equinox/internal/utility/fixed"
	"time"
)

type Configuration struct {
	LotValue         fixed.Point
	PipSize          fixed.Point
	CommissionPerLot fixed.Point
	PipSlippage      fixed.Point
	BarPeriod        time.Duration
	StartBalance     fixed.Point
}
