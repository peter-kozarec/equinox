package strategy

import (
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/model"
	"github.com/peter-kozarec/equinox/pkg/utility/circular"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"github.com/peter-kozarec/equinox/pkg/utility/math"
	"go.uber.org/zap"
	"time"
)

var (
	Three         = fixed.New(3, 0)
	NegativeThree = fixed.New(-3, 0)
)

type Advisor struct {
	logger *zap.Logger
	router *bus.Router

	lastTick model.Tick
	closes   *circular.Buffer[fixed.Point]
	zScores  *circular.Buffer[fixed.Point]

	posOpen bool
}

func NewAdvisor(logger *zap.Logger, router *bus.Router) *Advisor {
	return &Advisor{
		logger:  logger,
		router:  router,
		closes:  circular.NewBuffer[fixed.Point](60),
		zScores: circular.NewBuffer[fixed.Point](60),
		posOpen: false,
	}
}

func (a *Advisor) NewTick(t model.Tick) {
	a.lastTick = t
}

func (a *Advisor) NewBar(b model.Bar) {

	a.closes.Push(b.Close)

	if !a.closes.IsFull() {
		return
	}

	closes := a.closes.Data()
	mean := math.Mean(closes)
	stdDev := math.StandardDeviation(closes, mean)
	z := b.Close.Sub(mean).Div(stdDev)

	a.zScores.Push(z)

	if !a.zScores.IsFull() {
		return
	}

	if !a.posOpen && a.canTrade() {
		if z.Gte(Three) {
			_ = a.router.Post(bus.OrderEvent, model.Order{
				Command:    model.CmdOpen,
				OrderType:  model.Market,
				Size:       fixed.New(1, 2).Neg(),
				StopLoss:   b.Close.Add(b.Close.Sub(mean)),
				TakeProfit: mean,
			})
			a.posOpen = true
		} else if z.Lte(NegativeThree) {
			_ = a.router.Post(bus.OrderEvent, model.Order{
				Command:    model.CmdOpen,
				OrderType:  model.Market,
				Size:       fixed.New(1, 2),
				StopLoss:   b.Close.Sub(mean.Sub(b.Close)),
				TakeProfit: mean,
			})
			a.posOpen = true
		}
	}
}

func (a *Advisor) PositionClosed(_ model.Position) {
	a.posOpen = false
}

func (a *Advisor) canTrade() bool {

	if a.lastTick.TimeStamp == 0 {
		return false
	}

	t := time.Unix(0, a.lastTick.TimeStamp)

	if t.Hour() < 9 || t.Hour() > 18 {
		return false
	}

	return true
}
