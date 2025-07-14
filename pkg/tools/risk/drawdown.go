package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

var (
	lowDrawdownThreshold     = fixed.FromInt(2, 0)
	normalDrawdownThreshold  = fixed.FromInt(5, 0)
	highDrawdownThreshold    = fixed.FromInt(10, 0)
	extremeDrawdownThreshold = fixed.FromInt(15, 0)

	noDrawdownMultiplier      = fixed.FromFloat64(1.2)
	lowDrawdownMultiplier     = fixed.FromFloat64(1.0)
	normalDrawdownMultiplier  = fixed.FromFloat64(0.7)
	highDrawdownMultiplier    = fixed.FromFloat64(0.5)
	extremeDrawdownMultiplier = fixed.FromFloat64(0.0)
)

func withDrawdownCalcSize(baseSize fixed.Point, currentDrawdown fixed.Point) fixed.Point {
	if currentDrawdown.Lte(lowDrawdownThreshold) {
		return baseSize.Mul(noDrawdownMultiplier)
	} else if currentDrawdown.Lte(normalDrawdownThreshold) {
		return baseSize.Mul(lowDrawdownMultiplier)
	} else if currentDrawdown.Lte(highDrawdownThreshold) {
		return baseSize.Mul(normalDrawdownMultiplier)
	} else if currentDrawdown.Lte(extremeDrawdownThreshold) {
		return baseSize.Mul(highDrawdownMultiplier)
	} else {
		return baseSize.Mul(extremeDrawdownMultiplier)
	}
}
