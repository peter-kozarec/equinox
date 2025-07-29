package takeprofit

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type TakeProfit interface {
	GetInitialTakeProfit(common.Signal) (fixed.Point, error)
}
