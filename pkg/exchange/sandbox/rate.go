package sandbox

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"time"
)

type RateProvider interface {
	ExchangeRate(string, string, time.Time) (fixed.Point, error)
}
