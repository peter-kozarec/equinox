package simulation

import (
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"github.com/peter-kozarec/equinox/pkg/model"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
	"math/rand"
	"time"
)

var (
	pointFive = fixed.New(5, 1)
)

type MonteCarloExecutor struct {
	logger    *zap.Logger
	simulator *Simulator
	rng       *rand.Rand

	startTime  time.Time
	startPrice fixed.Point
	halfSpread fixed.Point
	mu         fixed.Point
	sigma      fixed.Point
	deltaT     fixed.Point
	steps      int64
	t          int64

	deltaLogPre1 fixed.Point
	deltaLogPre2 fixed.Point

	lastTime  time.Time
	lastPrice fixed.Point

	tick model.Tick
}

func NewMonteCarloExecutor(
	logger *zap.Logger,
	sim *Simulator,
	rng *rand.Rand,
	startTime time.Time,
	startPrice, fullSpread, mu, sigma, deltaT fixed.Point,
	steps int64) *MonteCarloExecutor {

	return &MonteCarloExecutor{
		logger:    logger,
		simulator: sim,
		rng:       rng,

		startTime:  startTime,
		startPrice: startPrice,
		halfSpread: fullSpread.DivInt(2),
		mu:         mu,
		sigma:      sigma,
		deltaT:     deltaT,
		steps:      steps,

		deltaLogPre1: mu.Sub(sigma.Mul(sigma).Mul(pointFive)).Mul(deltaT),
		deltaLogPre2: sigma.Mul(deltaT.Sqrt()),

		lastTime:  startTime,
		lastPrice: startPrice,
	}
}

func (e *MonteCarloExecutor) DoOnce() error {

	if e.t >= e.steps {
		return mapper.EOF
	}

	z := e.rng.NormFloat64()
	deltaLog := e.deltaLogPre1.Add(e.deltaLogPre2.Mul(fixed.FromFloat(z)))
	e.lastPrice = e.lastPrice.Mul(deltaLog.Exp())

	e.lastTime = e.lastTime.Add(time.Duration(333_000_000))
	e.t++

	e.tick.Ask = e.lastPrice.Add(e.halfSpread)
	e.tick.Bid = e.lastPrice.Sub(e.halfSpread)
	e.tick.TimeStamp = e.lastTime.UnixNano()
	e.tick.AskVolume = fixed.One
	e.tick.BidVolume = fixed.One

	if err := e.simulator.OnTick(e.tick); err != nil {
		return err
	}

	return nil
}
