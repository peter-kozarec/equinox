package common

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Signal struct {
	Entry    fixed.Point `json:"entry"`
	Target   fixed.Point `json:"target"`
	Strength uint8       `json:"strength,omitempty"`
	Comment  string      `json:"comment,omitempty"`

	Source      string              `json:"src,omitempty"`
	Symbol      string              `json:"symbol,omitempty"`
	ExecutionID utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}

type SignalRejected struct {
	Reason         string `json:"reason"`
	Comment        string `json:"comment,omitempty"`
	OriginalSignal Signal `json:"original_signal"`

	Source      string              `json:"src,omitempty"`
	ExecutionID utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}

type SignalAccepted struct {
	Comment        string `json:"comment,omitempty"`
	OriginalSignal Signal `json:"original_signal"`

	Source      string              `json:"src,omitempty"`
	ExecutionID utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts"`
}
