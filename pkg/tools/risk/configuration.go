package risk

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Configuration struct {
	RiskMax  fixed.Point
	RiskMin  fixed.Point
	RiskBase fixed.Point
	RiskOpen fixed.Point
}
