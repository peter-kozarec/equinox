package math

import (
	"peter-kozarec/equinox/pkg/utility/fixed"
)

func Mean(returns []fixed.Point) fixed.Point {
	var sum fixed.Point
	for _, r := range returns {
		sum = sum.Add(r)
	}
	return sum.DivInt(len(returns))
}
