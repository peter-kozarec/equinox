package math

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func SharpeRatio(returns []fixed.Point, riskFreeRate fixed.Point) fixed.Point {
	mean := Mean(returns)
	volatility := StandardDeviation(returns, mean)
	return mean.Sub(riskFreeRate).Div(volatility)
}
