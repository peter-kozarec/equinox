package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Bar struct {
	Source      string              `json:"src,omitempty"`
	Symbol      string              `json:"symbol,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
	Period      time.Duration       `json:"period"`
	Open        fixed.Point         `json:"open"`
	High        fixed.Point         `json:"high"`
	Low         fixed.Point         `json:"low"`
	Close       fixed.Point         `json:"close"`
	Volume      fixed.Point         `json:"volume"`
}
