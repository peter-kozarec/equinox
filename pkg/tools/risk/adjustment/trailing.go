package adjustment

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type TrailingStopLoss struct {
	trailingDistance fixed.Point
	trailingTreshold fixed.Point
}

func NewTrailingStopLoss(trailingDistance, trailingTreshold fixed.Point) *TrailingStopLoss {
	return &TrailingStopLoss{
		trailingDistance: trailingDistance,
		trailingTreshold: trailingTreshold,
	}
}

func (t *TrailingStopLoss) AdjustPosition(position common.Position, tick common.Tick) (fixed.Point, fixed.Point, bool) {
	if t.trailingDistance.Lte(fixed.Zero) || t.trailingTreshold.Lte(fixed.Zero) {
		return fixed.Zero, fixed.Zero, false
	}
	if position.StopLoss.IsZero() || position.TakeProfit.IsZero() {
		return fixed.Zero, fixed.Zero, false
	}

	var newStopLoss, newTakeProfit fixed.Point
	var triggered bool

	if position.Side == common.PositionSideLong {
		openPrice := position.OpenPrice
		if position.StopLoss.Gte(openPrice) {
			openPrice = position.StopLoss
		}
		priceThreshold := openPrice.Add(openPrice.Mul(fixed.One.Add(t.trailingTreshold)))
		if tick.Bid.Gte(priceThreshold) {
			triggered = true
			newStopLoss = priceThreshold.Sub(priceThreshold.Mul(fixed.One.Add(t.trailingDistance)))
			newTakeProfit = position.TakeProfit.Add(newStopLoss.Sub(openPrice))
		}
	} else {
		openPrice := position.OpenPrice
		if position.StopLoss.Lte(openPrice) {
			openPrice = position.StopLoss
		}
		priceThreshold := openPrice.Sub(openPrice.Mul(fixed.One.Add(t.trailingTreshold)))
		if tick.Ask.Lte(priceThreshold) {
			triggered = true
			newStopLoss = priceThreshold.Add(priceThreshold.Mul(fixed.One.Add(t.trailingDistance)))
			newTakeProfit = position.TakeProfit.Sub(openPrice.Sub(newStopLoss))
		}
	}

	return newStopLoss, newTakeProfit, triggered
}
