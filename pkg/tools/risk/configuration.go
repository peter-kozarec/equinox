package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type Configuration struct {
	BaseRiskPerTrade        fixed.Point
	MaxRiskPerTrade         fixed.Point
	MaxRiskForAllOpenTrades fixed.Point

	ValuePerLot fixed.Point

	SizeScale  int
	PriceScale int

	AtrWindowSize int

	LowVolatilityAtrThreshold  fixed.Point
	HighVolatilityAtrThreshold fixed.Point

	LowVolatilityStopLossAtrMultiplier    fixed.Point
	NormalVolatilityStopLossAtrMultiplier fixed.Point
	HighVolatilityStopLossAtrMultiplier   fixed.Point
}

var (
	DefaultConfiguration = Configuration{
		BaseRiskPerTrade:                      fixed.FromInt64(1, 0),
		MaxRiskPerTrade:                       fixed.FromInt64(2, 0),
		MaxRiskForAllOpenTrades:               fixed.FromInt64(5, 0),
		ValuePerLot:                           fixed.FromInt64(100000, 0),
		SizeScale:                             2,
		PriceScale:                            5,
		AtrWindowSize:                         144,
		LowVolatilityAtrThreshold:             fixed.FromInt64(10, 0),
		HighVolatilityAtrThreshold:            fixed.FromInt64(20, 0),
		LowVolatilityStopLossAtrMultiplier:    fixed.FromInt64(15, 1),
		NormalVolatilityStopLossAtrMultiplier: fixed.FromInt64(1, 0),
		HighVolatilityStopLossAtrMultiplier:   fixed.FromInt64(5, 1),
	}
)
