package risk

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Configuration struct {
	RiskMax  fixed.Point
	RiskMin  fixed.Point
	RiskBase fixed.Point
	RiskOpen fixed.Point

	AtrPeriod                  int
	AtrStopLossMultiplier      fixed.Point
	AtrTakeProfitMinMultiplier fixed.Point

	BreakEvenMove      fixed.Point
	BreakEvenThreshold fixed.Point

	Cooldown time.Duration
}
