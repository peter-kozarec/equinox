package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Signal struct {
	Source      string              `json:"source,omitempty"`
	Symbol      string              `json:"symbol,omitempty"`
	ExecutionID utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
	Entry       fixed.Point         `json:"entry"`
	Target      fixed.Point         `json:"target"`
	Strength    uint8               `json:"strength,omitempty"`
	Comment     string              `json:"comment,omitempty"`
}
