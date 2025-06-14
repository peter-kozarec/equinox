package math

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func SortinoRatio(returns []fixed.Point, riskFreeRate fixed.Point) fixed.Point {
	mean := Mean(returns)
	downsideDeviation := DownsideDeviation(returns, riskFreeRate)
	return mean.Sub(riskFreeRate).Div(downsideDeviation)
}
