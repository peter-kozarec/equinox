package math

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func SharpeRatio(data []fixed.Point, riskFreeRate fixed.Point) fixed.Point {
	mean := Mean(data)
	volatility := StandardDeviation(data, mean)
	return mean.Sub(riskFreeRate).Div(volatility)
}
