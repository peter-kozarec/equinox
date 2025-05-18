package model

import (
	"peter-kozarec/equinox/internal/utility"
	"time"
)

type Bar struct {
	Period    time.Duration
	TimeStamp int64
	Open      utility.Fixed
	High      utility.Fixed
	Low       utility.Fixed
	Close     utility.Fixed
	Volume    int32
}
