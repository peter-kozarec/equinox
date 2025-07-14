package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

const (
	highSignalStrengthThreshold   uint8 = 90
	mediumSignalStrengthThreshold uint8 = 70
	lowSignalStrengthThreshold    uint8 = 50
)

var (
	highSignalStrengthMultiplier   = fixed.FromFloat64(1.0)
	mediumSignalStrengthMultiplier = fixed.FromFloat64(0.7)
	lowSignalStrengthMultiplier    = fixed.FromFloat64(0.5)
	noSignalStrengthMultiplier     = fixed.FromFloat64(0.0)
)

func withSignalStrengthCalcSize(baseSize fixed.Point, signalStrength uint8) fixed.Point {
	if signalStrength >= highSignalStrengthThreshold {
		return baseSize.Mul(highSignalStrengthMultiplier)
	} else if signalStrength >= mediumSignalStrengthThreshold {
		return baseSize.Mul(mediumSignalStrengthMultiplier)
	} else if signalStrength >= lowSignalStrengthThreshold {
		return baseSize.Mul(lowSignalStrengthMultiplier)
	} else {
		return baseSize.Mul(noSignalStrengthMultiplier)
	}
}
