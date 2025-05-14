package model

import "github.com/govalues/decimal"

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       float64
	Bid       float64
	AskVolume float64
	BidVolume float64
}

func (tick *Tick) AskDecimal() (decimal.Decimal, error) {
	return decimal.NewFromFloat64(tick.Ask)
}

func (tick *Tick) BidDecimal() (decimal.Decimal, error) {
	return decimal.NewFromFloat64(tick.Bid)
}
