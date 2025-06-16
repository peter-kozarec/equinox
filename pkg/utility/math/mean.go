package math

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func Mean(data []fixed.Point) fixed.Point {
	var sum fixed.Point
	for _, r := range data {
		sum = sum.Add(r)
	}
	return sum.DivInt(len(data))
}
