package model

import (
	"time"
)

type Bar struct {
	Period    time.Duration
	TimeStamp int64
	Open      Price
	High      Price
	Low       Price
	Close     Price
	Volume    Price
}
