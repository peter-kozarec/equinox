package math

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func SortinoRatio(data []fixed.Point, riskFreeRate fixed.Point) fixed.Point {
	mean := Mean(data)
	downsideDeviation := DownsideDeviation(data, riskFreeRate)
	return mean.Sub(riskFreeRate).Div(downsideDeviation)
}
