package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type BarPeriod = time.Duration

const (
	BarPeriodM1  = time.Minute
	BarPeriodM5  = time.Minute * 5
	BarPeriodM10 = time.Minute * 10
	BarPeriodM15 = time.Minute * 15
	BarPeriodM30 = time.Minute * 30
	BarPeriodH1  = time.Hour
)

type Bar struct {
	OpenTime time.Time   `json:"open_time"`
	Period   BarPeriod   `json:"period"`
	Open     fixed.Point `json:"open"`
	High     fixed.Point `json:"high"`
	Low      fixed.Point `json:"low"`
	Close    fixed.Point `json:"close"`
	Volume   fixed.Point `json:"volume"`

	Source      string              `json:"src,omitempty"`
	Symbol      string              `json:"symbol,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}
