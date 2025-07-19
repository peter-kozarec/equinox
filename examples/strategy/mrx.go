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
}

func NewMrxAdvisor(router *bus.Router) *MrxAdvisor {
	return &MrxAdvisor{
		router:  router,
		closes:  fixed.NewRingBuffer(60),
		zScores: fixed.NewRingBuffer(60),
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

	if z.Gte(Three) || z.Lte(NegativeThree) {
		_ = a.router.Post(bus.SignalEvent, common.Signal{
			Source:      componentName,
			Symbol:      b.Symbol,
			ExecutionID: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   b.TimeStamp,
			Entry:       a.lastTick.Bid,
			Target:      mean,
			Strength:    60,
			Comment:     z.String(),
		})
	}
}
