package math

import (
	"peter-kozarec/equinox/pkg/utility/fixed"
)

func DownsideDeviation(returns []fixed.Point, riskFreeRate fixed.Point) fixed.Point {
	var sum fixed.Point
	var count int
	for _, r := range returns {
		if r.Lt(riskFreeRate) {
			diff := r.Sub(riskFreeRate)
			sum = sum.Add(diff.Mul(diff))
			count++
		}
	}
	if count == 0 {
		return fixed.Zero
	}
	return sum.DivInt(count).Sqrt()
}

func StandardDeviation(returns []fixed.Point, mean fixed.Point) fixed.Point {
	var sum fixed.Point
	for _, r := range returns {
		diff := r.Sub(mean)
		sum = sum.Add(diff.Mul(diff))
	}
	return sum.DivInt(len(returns)).Sqrt()
}
