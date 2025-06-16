package math

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func DownsideDeviation(data []fixed.Point, riskFreeRate fixed.Point) fixed.Point {
	var sum fixed.Point
	var count int
	for _, r := range data {
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

func StandardDeviation(data []fixed.Point, mean fixed.Point) fixed.Point {
	var sum fixed.Point
	for _, r := range data {
		diff := r.Sub(mean)
		sum = sum.Add(diff.Mul(diff))
	}
	return sum.DivInt(len(data)).Sqrt()
}
