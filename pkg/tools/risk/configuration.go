package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type Configuration struct {
	// Position size
	MaxRiskPercentage  fixed.Point
	MinRiskPercentage  fixed.Point
	BaseRiskPercentage fixed.Point

	// Max risk
	MaxOpenRiskPercentage fixed.Point

	// Stop Loss
	AtrPeriod       int
	SlAtrMultiplier fixed.Point

	// Break Even - percentage of what price has to move towards the take profit to change the stop loss to break even
	BreakEvenMovePercentage      fixed.Point // Where stop loss will be moved
	BreakEvenThresholdPercentage fixed.Point // Threshold to move stop loss
}
