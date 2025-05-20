package calc

import "peter-kozarec/equinox/internal/utility"

func Mean(returns []utility.Fixed) utility.Fixed {
	var sum utility.Fixed
	for _, r := range returns {
		sum = sum.Add(r)
	}
	return sum.DivInt(len(returns))
}
