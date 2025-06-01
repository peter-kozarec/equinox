package calc

import (
	"peter-kozarec/equinox/internal/utility/fixed"
)

func SharpeRatio(returns []fixed.Point, riskFreeRate fixed.Point) fixed.Point {
	mean := Mean(returns)
	volatility := StandardDeviation(returns, mean)
	return mean.Sub(riskFreeRate).Div(volatility)
}
