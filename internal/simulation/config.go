package simulation

import "time"

const (
	StartingBalance          = 1000000
	StartingBalancePrecision = 2

	Slippage          = 15
	SlippagePrecision = 1

	Commission          = 3
	CommissionPrecision = 0

	LotValue          = 100000
	LotValuePrecision = 0
	PipSize           = 1
	PipSizePrecision  = 4

	InstrumentPrecision = 5

	AccountSnapshotInterval = time.Minute
	BarPeriod               = time.Minute
)
