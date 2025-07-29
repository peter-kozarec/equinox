package adjustment

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type BreakEven struct {
	moveThreshold fixed.Point
	moveDistance  fixed.Point
}

func NewBreakEven(moveThreshold, moveDistance fixed.Point) *BreakEven {
	return &BreakEven{
		moveThreshold: moveThreshold,
		moveDistance:  moveDistance,
	}
}

func (b *BreakEven) AdjustPosition(position common.Position, tick common.Tick) (fixed.Point, fixed.Point, bool) {
	if b.isBreakEvenAlreadySet(position) {
		return fixed.Zero, fixed.Zero, false
	}

	movePercentage := b.calculateMovePercentage(position, tick)
	if movePercentage.Lt(b.moveThreshold) {
		return fixed.Zero, fixed.Zero, false
	}

	newStopLoss := b.calculateBreakEvenStopLoss(position)
	return newStopLoss, position.TakeProfit, true
}

func (b *BreakEven) isBreakEvenAlreadySet(position common.Position) bool {
	if position.Side == common.PositionSideLong {
		return position.StopLoss.Gte(position.OpenPrice)
	}
	return position.StopLoss.Lte(position.OpenPrice)
}

func (b *BreakEven) calculateMovePercentage(position common.Position, tick common.Tick) fixed.Point {
	var moved, takeProfitPriceDiff fixed.Point

	if position.Side == common.PositionSideLong {
		if position.TakeProfit.Lte(tick.Bid) {
			return fixed.FromInt(100, 0)
		}
		moved = tick.Bid.Sub(position.OpenPrice)
		takeProfitPriceDiff = position.TakeProfit.Sub(position.OpenPrice)
	} else {
		if position.TakeProfit.Gte(tick.Ask) {
			return fixed.FromInt(100, 0)
		}
		moved = position.OpenPrice.Sub(tick.Ask)
		takeProfitPriceDiff = position.OpenPrice.Sub(position.TakeProfit)
	}

	if moved.Lt(fixed.Zero) {
		return fixed.Zero
	}

	return moved.Div(takeProfitPriceDiff).MulInt(100)
}

func (b *BreakEven) calculateBreakEvenStopLoss(position common.Position) fixed.Point {
	takeProfitPriceDiff := position.TakeProfit.Sub(position.OpenPrice).Abs()
	newStopLossMove := takeProfitPriceDiff.Mul(b.moveDistance.DivInt(100))

	if position.Side == common.PositionSideLong {
		return position.OpenPrice.Add(newStopLossMove)
	}
	return position.OpenPrice.Sub(newStopLossMove)
}
