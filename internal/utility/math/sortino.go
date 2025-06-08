package math

import (
	"peter-kozarec/equinox/internal/utility/fixed"
)

func SortinoRatio(returns []fixed.Point, riskFreeRate fixed.Point) fixed.Point {
	mean := Mean(returns)
	downsideDeviation := DownsideDeviation(returns, riskFreeRate)
	return mean.Sub(riskFreeRate).Div(downsideDeviation)
}
