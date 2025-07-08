package strategy

import (
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"

	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/circular"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	Three         = fixed.FromInt64(3, 0)
	NegativeThree = fixed.FromInt64(-3, 0)
)

// MrxAdvisor is a test strategy, not meant for production
type MrxAdvisor struct {
	router *bus.Router

	lastTick common.Tick
	closes   *circular.Buffer[fixed.Point]
	zScores  *circular.Buffer[fixed.Point]

	posOpen bool
}

func NewMrxAdvisor(router *bus.Router) *MrxAdvisor {
	return &MrxAdvisor{
		router:  router,
		closes:  circular.NewBuffer[fixed.Point](60),
		zScores: circular.NewBuffer[fixed.Point](60),
		posOpen: false,
	}
}

func (a *MrxAdvisor) NewTick(t common.Tick) {
	a.lastTick = t
}

func (a *MrxAdvisor) NewBar(b common.Bar) {

	a.closes.Push(b.Close)

	if !a.closes.IsFull() {
		return
	}

	closes := a.closes.Data()
	mean := fixed.Mean(closes)
	stdDev := fixed.StdDev(closes, mean)
	z := b.Close.Sub(mean).Div(stdDev)

	a.zScores.Push(z)

	if !a.zScores.IsFull() {
		return
	}

	if !a.posOpen && a.canTrade() {
		if z.Gte(Three) {
			_ = a.router.Post(bus.OrderEvent, common.Order{
				Command:    common.CmdOpen,
				OrderType:  common.Market,
				Size:       fixed.FromInt64(1, 2).Neg(),
				StopLoss:   b.Close.Add(b.Close.Sub(mean)),
				TakeProfit: mean,
			})
			a.posOpen = true
		} else if z.Lte(NegativeThree) {
			_ = a.router.Post(bus.OrderEvent, common.Order{
				Command:    common.CmdOpen,
				OrderType:  common.Market,
				Size:       fixed.FromInt64(1, 2),
				StopLoss:   b.Close.Sub(mean.Sub(b.Close)),
				TakeProfit: mean,
			})
			a.posOpen = true
		}
	}
}

func (a *MrxAdvisor) PositionClosed(_ common.Position) {
	a.posOpen = false
}

func (a *MrxAdvisor) canTrade() bool {

	if a.lastTick.TimeStamp == 0 {
		return false
	}

	t := time.Unix(0, a.lastTick.TimeStamp)

	if t.Hour() < 9 || t.Hour() > 18 {
		return false
	}

	return true
}
