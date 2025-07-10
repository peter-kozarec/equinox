package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"time"
)

type Balance struct {
	Source      string              `json:"src,omitempty"`
	Account     string              `json:"account,omitempty"`
	ExecutionId utility.ExecutionID `json:"eid,omitempty"`
	TraceID     utility.TraceID     `json:"tid,omitempty"`
	TimeStamp   time.Time           `json:"ts,omitempty"`
	Value       fixed.Point         `json:"value"`
}
