package calc

import "peter-kozarec/equinox/internal/utility"

func SortinoRatio(returns []utility.Fixed, riskFreeRate utility.Fixed) utility.Fixed {
	mean := Mean(returns)
	downsideDeviation := DownsideDeviation(returns, riskFreeRate)
	return mean.Sub(riskFreeRate).Div(downsideDeviation)
}
