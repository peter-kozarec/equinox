package risk

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/tools/indicators"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	ErrSLAtrNotReady = errors.New("atr for stop loss is not ready")
)

type StopLossHandler interface {
	CalcStopLoss(common.Signal) (fixed.Point, error)
}

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

func (a *AtrBasedStopLoss) CalcStopLoss(signal common.Signal) (fixed.Point, error) {
	if !a.atr.Ready() {
		return fixed.Point{}, ErrSLAtrNotReady
	}
	atrValue := a.atr.Value()
	if signal.Target.Gt(signal.Entry) {
		return signal.Entry.Sub(atrValue.Mul(a.atrMultiplier)), nil
	}
	return signal.Entry.Add(atrValue.Mul(a.atrMultiplier)), nil
}
