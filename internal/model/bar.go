package model

import (
	"time"
)

type Bar struct {
	Period    time.Duration
	TimeStamp int64
	Open      int32
	High      int32
	Low       int32
	Close     int32
	Volume    int32
}
