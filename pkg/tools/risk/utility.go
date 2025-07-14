package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

func clamp(p, min, max fixed.Point) fixed.Point {
	if p.Gt(max) {
		return max
	} else if p.Lt(min) {
		return min
	} else {
		return p
	}
}
