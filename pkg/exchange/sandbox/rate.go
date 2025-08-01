package sandbox

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type RateProvider interface {
	ExchangeRate(string, string, time.Time) (fixed.Point, fixed.Point, error)
}
