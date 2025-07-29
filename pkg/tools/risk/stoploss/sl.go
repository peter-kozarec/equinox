package stoploss

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type StopLoss interface {
	GetInitialStopLoss(common.Signal) (fixed.Point, error)
}
