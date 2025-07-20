package strategy

import (
	"context"
	"fmt"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	mrxComponentName = "example.strategy.mrx"
)

var (
	PositiveThreshold = fixed.FromInt64(4, 0)
	NegativeThreshold = PositiveThreshold.Neg()
)

type MeanReversion struct {
	router *bus.Router

	tick    common.Tick
	closes  *fixed.RingBuffer
	zScores *fixed.RingBuffer
}

func NewMeanReversion(router *bus.Router, window int) *MeanReversion {
	return &MeanReversion{
		router:  router,
		closes:  fixed.NewRingBuffer(window),
		zScores: fixed.NewRingBuffer(window),
	}
}

func (m *MeanReversion) OnTick(_ context.Context, tick common.Tick) {
	m.tick = tick
}

func (m *MeanReversion) OnBar(_ context.Context, bar common.Bar) {
	m.closes.Add(bar.Close)

	if m.closes.IsFull() {
		mean := m.closes.Mean()
		stdDev := m.closes.SampleStdDev()
		z := bar.Close.Sub(mean).Div(stdDev)
		m.zScores.Add(z)

		if m.zScores.IsFull() {
			if z.Gte(PositiveThreshold) || z.Lte(NegativeThreshold) {
				_ = m.router.Post(bus.SignalEvent, common.Signal{
					Source:      mrxComponentName,
					Symbol:      bar.Symbol,
					ExecutionID: utility.GetExecutionID(),
					TraceID:     utility.CreateTraceID(),
					TimeStamp:   bar.TimeStamp,
					Entry:       m.tick.Bid.Add(m.tick.Ask).DivInt(2),
					Target:      mean,
					Strength:    100,
					Comment:     fmt.Sprintf("z-score: %v", z),
				})
			}
		}
	}
}
