package calc

import (
	"peter-kozarec/equinox/internal/utility"
)

func DownsideDeviation(returns []utility.Fixed, riskFreeRate utility.Fixed) utility.Fixed {
	var sum utility.Fixed
	var count int
	for _, r := range returns {
		if r.Lt(riskFreeRate) {
			diff := r.Sub(riskFreeRate)
			sum = sum.Add(diff.Mul(diff))
			count++
		}
	}
	if count == 0 {
		return utility.ZeroFixed
	}
	return sum.DivInt(count).Sqrt()
}

func StandardDeviation(returns []utility.Fixed, mean utility.Fixed) utility.Fixed {
	var sum utility.Fixed
	for _, r := range returns {
		diff := r.Sub(mean)
		sum = sum.Add(diff.Mul(diff))
	}
	return sum.DivInt(len(returns)).Sqrt()
}
