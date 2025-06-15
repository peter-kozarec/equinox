package main

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

func autocorrelation(data []fixed.Point, mean fixed.Point, lag int) fixed.Point {
	n := len(data)
	if lag >= n {
		panic("lag must be less than data length")
	}

	var numerator, denominator fixed.Point
	for i := 0; i < n-lag; i++ {
		numerator = numerator.Add(data[i].Sub(mean).Mul(data[i+lag].Sub(mean)))
	}
	for i := 0; i < n; i++ {
		diff := data[i].Sub(mean)
		denominator = denominator.Add(diff.Mul(diff))
	}

	cov := numerator.DivInt(n - lag)
	return cov.Div(denominator.DivInt(n))
}
