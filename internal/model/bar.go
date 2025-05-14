package model

import (
	"github.com/govalues/decimal"
	"time"
)

type Bar struct {
	Period    time.Duration
	TimeStamp int64
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Volume    float64
}

func (bar *Bar) OpenDecimal() (decimal.Decimal, error) {
	return decimal.NewFromFloat64(bar.Open)
}

func (bar *Bar) CloseDecimal() (decimal.Decimal, error) {
	return decimal.NewFromFloat64(bar.Close)
}

func (bar *Bar) HighDecimal() (decimal.Decimal, error) {
	return decimal.NewFromFloat64(bar.High)
}

func (bar *Bar) LowDecimal() (decimal.Decimal, error) {
	return decimal.NewFromFloat64(bar.Low)
}
