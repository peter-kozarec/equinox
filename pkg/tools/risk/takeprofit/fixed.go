package takeprofit

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type FixedTakeProfit struct {
}

func NewFixedTakeProfit() *FixedTakeProfit {
	return &FixedTakeProfit{}
}

func (f *FixedTakeProfit) GetInitialTakeProfit(signal common.Signal) (fixed.Point, error) {
	return signal.Target, nil
}
