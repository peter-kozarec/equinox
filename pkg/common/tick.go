package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"time"
)

type Tick struct {
	Ask       fixed.Point `json:"ask"`
	Bid       fixed.Point `json:"bid"`
	AskVolume fixed.Point `json:"ask_volume"`
	BidVolume fixed.Point `json:"bid_volume"`

	Source      string              `json:"src,omitempty"`
	Symbol      string              `json:"symbol,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}
