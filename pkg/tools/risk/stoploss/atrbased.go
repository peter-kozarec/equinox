package stoploss

import (
	"context"
	"fmt"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/tools/indicators"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type AtrBasedStopLoss struct {
	atr           *indicators.Atr
	atrMultiplier fixed.Point
}

func NewAtrBasedStopLoss(atrWindow int, atrMultiplier fixed.Point) *AtrBasedStopLoss {
	return &AtrBasedStopLoss{
		atr:           indicators.NewAtr(atrWindow),
		atrMultiplier: atrMultiplier,
	}
}

func (a *AtrBasedStopLoss) OnBar(_ context.Context, bar common.Bar) {
	a.atr.OnBar(bar)
}

func (a *AtrBasedStopLoss) GetInitialStopLoss(signal common.Signal) (fixed.Point, error) {
	if !a.atr.Ready() {
		return fixed.Zero, fmt.Errorf("atr is not ready")
	}
	atrVal := a.atr.Value()
	if signal.Entry.Gt(signal.Target) {
		return signal.Entry.Add(atrVal.Mul(a.atrMultiplier).Abs()), nil
	}
	return signal.Entry.Sub(atrVal.Mul(a.atrMultiplier).Abs()), nil
}
