package model

import "github.com/govalues/decimal"

type Tick struct {
	TimeStamp int64 // Unix NanoSeconds
	Ask       float64
	Bid       float64
	AskVolume float64
	BidVolume float64
}

func (t *Tick) AskDecimal() (decimal.Decimal, error) {
	return decimal.NewFromFloat64(t.Ask)
}

func (t *Tick) BidDecimal() (decimal.Decimal, error) {
	return decimal.NewFromFloat64(t.Bid)
}
