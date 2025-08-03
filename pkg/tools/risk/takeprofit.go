package risk

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/tools/indicators"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	ErrTPAtrNotReady = errors.New("atr for take profit is not ready")
)

type TakeProfitHandler interface {
	CalcTakeProfit(common.Signal) (fixed.Point, error)
}

type FixedTakeProfit struct{}

func NewFixedTakeProfit() *FixedTakeProfit {
	return &FixedTakeProfit{}
}

func (f *FixedTakeProfit) CalcTakeProfit(signal common.Signal) (fixed.Point, error) {
	return signal.Target, nil
}

type AtrBasedTakeProfit struct {
	atr           *indicators.Atr
	atrMultiplier fixed.Point
}

func NewAtrBasedTakeProfit(atrWindow int, atrMultiplier fixed.Point) *AtrBasedTakeProfit {
	return &AtrBasedTakeProfit{
		atr:           indicators.NewAtr(atrWindow),
		atrMultiplier: atrMultiplier,
	}
}

func (a *AtrBasedTakeProfit) OnBar(_ context.Context, bar common.Bar) {
	a.atr.OnBar(bar)
}

func (a *AtrBasedTakeProfit) CalcTakeProfit(signal common.Signal) (fixed.Point, error) {
	if !a.atr.Ready() {
		return fixed.Point{}, ErrTPAtrNotReady
	}
	atrValue := a.atr.Value()
	if signal.Target.Gt(signal.Entry) {
		return signal.Entry.Add(atrValue.Mul(a.atrMultiplier)), nil
	}
	return signal.Entry.Sub(atrValue.Mul(a.atrMultiplier)), nil
}
