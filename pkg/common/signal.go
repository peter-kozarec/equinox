package common

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Signal struct {
	Source    string // Signal source identifier
	Entry     fixed.Point
	Target    fixed.Point
	Strength  uint8         // Signal strength (0-100)
	TimeFrame time.Duration // Bar timeframe (0 for tick)
	Comment   string        // Additional comment about the signal
}
