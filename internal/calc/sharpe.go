package calc

import "peter-kozarec/equinox/internal/utility"

func SharpeRatio(returns []utility.Fixed, riskFreeRate utility.Fixed) utility.Fixed {
	mean := Mean(returns)
	volatility := StandardDeviation(returns, mean)
	return mean.Sub(riskFreeRate).Div(volatility)
}
