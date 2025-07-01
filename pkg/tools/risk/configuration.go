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
		BaseRiskPerTrade:                      fixed.New(1, 0),
		MaxRiskPerTrade:                       fixed.New(2, 0),
		MaxRiskForAllOpenTrades:               fixed.New(5, 0),
		ValuePerLot:                           fixed.New(100000, 0),
		SizeScale:                             2,
		PriceScale:                            5,
		AtrWindowSize:                         144,
		LowVolatilityAtrThreshold:             fixed.New(10, 0),
		HighVolatilityAtrThreshold:            fixed.New(20, 0),
		LowVolatilityStopLossAtrMultiplier:    fixed.New(15, 1),
		NormalVolatilityStopLossAtrMultiplier: fixed.New(1, 0),
		HighVolatilityStopLossAtrMultiplier:   fixed.New(5, 1),
	}
)
