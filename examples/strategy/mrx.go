package strategy

import (
	"context"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	componentName = "mrx"
)

var (
	Three         = fixed.FromInt64(3, 0)
	NegativeThree = fixed.FromInt64(-3, 0)
)

// MrxAdvisor is a test strategy, not meant for production
type MrxAdvisor struct {
	router *bus.Router

	lastTick common.Tick
	closes   *fixed.RingBuffer
	zScores  *fixed.RingBuffer

	posOpen bool
}

func NewMrxAdvisor(router *bus.Router) *MrxAdvisor {
	return &MrxAdvisor{
		router:  router,
		closes:  fixed.NewRingBuffer(60),
		zScores: fixed.NewRingBuffer(60),
		posOpen: false,
	}
}

func (a *MrxAdvisor) OnTick(_ context.Context, t common.Tick) {
	a.lastTick = t
}

func (a *MrxAdvisor) OnBar(_ context.Context, b common.Bar) {

	a.closes.Add(b.Close)

	if !a.closes.IsFull() {
		return
	}

	mean := a.closes.Mean()
	stdDev := a.closes.SampleStdDev()
	z := b.Close.Sub(mean).Div(stdDev)

	a.zScores.Add(z)

	if !a.zScores.IsFull() {
		return
	}

	if !a.posOpen && a.canTrade() {
		if z.Gte(Three) {
			_ = a.router.Post(bus.OrderEvent, common.Order{
				Source:      componentName,
				Symbol:      b.Symbol,
				ExecutionId: utility.GetExecutionID(),
				TraceID:     utility.CreateTraceID(),
				Side:        common.OrderSideSell,
				TimeStamp:   b.TimeStamp,
				Command:     common.OrderCommandPositionOpen,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromInt64(1, 2).Neg(),
				StopLoss:    b.Close.Add(b.Close.Sub(mean)),
				TakeProfit:  mean,
			})
			a.posOpen = true
		} else if z.Lte(NegativeThree) {
			_ = a.router.Post(bus.OrderEvent, common.Order{
				Source:      componentName,
				Symbol:      b.Symbol,
				ExecutionId: utility.GetExecutionID(),
				TraceID:     utility.CreateTraceID(),
				Command:     common.OrderCommandPositionOpen,
				Type:        common.OrderTypeMarket,
				Side:        common.OrderSideBuy,
				Size:        fixed.FromInt64(1, 2),
				StopLoss:    b.Close.Sub(mean.Sub(b.Close)),
				TakeProfit:  mean,
			})
			a.posOpen = true
		}
	}
}

func (a *MrxAdvisor) OnPositionClosed(_ context.Context, _ common.Position) {
	a.posOpen = false
}

func (a *MrxAdvisor) canTrade() bool {

	if a.lastTick.TimeStamp.IsZero() {
		return false
	}

	if a.lastTick.TimeStamp.Hour() < 9 || a.lastTick.TimeStamp.Hour() > 18 {
		return false
	}

	return true
}
