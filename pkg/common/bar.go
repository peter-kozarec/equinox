package common

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Bar struct {
	Period    time.Duration
	TimeStamp int64
	Open      fixed.Point
	High      fixed.Point
	Low       fixed.Point
	Close     fixed.Point
	Volume    fixed.Point
}
