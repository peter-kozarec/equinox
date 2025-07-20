package synthetic

import (
	"errors"
	"math/rand"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	tickGeneratorComponentName = "datasource.synthetic.generator"
)

var (
	pointFive = fixed.FromInt64(5, 1)
	ErrEof    = errors.New("EOF")
)

type TickGenerator struct {
	symbol string
	rng    *rand.Rand

	startTime  time.Time
	startPrice fixed.Point
	baseSpread fixed.Point
	mu         fixed.Point
	sigma      fixed.Point
	deltaT     fixed.Point
	steps      int64
	t          int64

	avgTickInterval time.Duration
	tickVariability float64

	avgVolume      fixed.Point
	volumeVariance float64

	deltaLogPre1 fixed.Point
	deltaLogPre2 fixed.Point

	spreadVolatility float64
	minSpread        fixed.Point
	maxSpread        fixed.Point

	lastTime      time.Time
	lastPrice     fixed.Point
	currentSpread fixed.Point

	normPriceDigits  int
	normVolumeDigits int
}

func NewTickGenerator(
	symbol string,
	rng *rand.Rand,
	startTime time.Time,
	startPrice, fullSpread, mu, sigma, deltaT fixed.Point,
	steps int64) *TickGenerator {

	return &TickGenerator{
		symbol: symbol,
		rng:    rng,

		startTime:  startTime,
		startPrice: startPrice,
		baseSpread: fullSpread.DivInt64(2),
		mu:         mu,
		sigma:      sigma,
		deltaT:     deltaT,
		steps:      steps,

		avgTickInterval: time.Duration(333_000_000),
		tickVariability: 0.3, // 30% variation in tick timing

		avgVolume:      fixed.FromInt64(100, 0), // Average 100 units
		volumeVariance: 0.5,                     // 50% variance in volume

		spreadVolatility: 0.1,
		minSpread:        fullSpread.Mul(fixed.FromInt64(5, 1)),  // 50% of base spread
		maxSpread:        fullSpread.Mul(fixed.FromInt64(15, 1)), // 150% of base spread

		// Pre-calculated values for GBM
		deltaLogPre1: mu.Sub(sigma.Mul(sigma).Mul(pointFive)).Mul(deltaT),
		deltaLogPre2: sigma.Mul(deltaT.Sqrt()),

		lastTime:      startTime,
		lastPrice:     startPrice,
		currentSpread: fullSpread.DivInt64(2),
	}
}

func (e *TickGenerator) SetTickParameters(avgInterval time.Duration, intervalVariability float64, avgVol fixed.Point, volVariance float64) {
	e.avgTickInterval = avgInterval
	e.tickVariability = intervalVariability
	e.avgVolume = avgVol
	e.volumeVariance = volVariance
}

func (e *TickGenerator) SetSpreadDynamics(volatility float64, minSpread, maxSpread fixed.Point) {
	e.spreadVolatility = volatility
	e.minSpread = minSpread
	e.maxSpread = maxSpread
}

func (e *TickGenerator) SetPriceDigits(digits int) {
	e.normPriceDigits = digits
}

func (e *TickGenerator) SetVolumeDigits(digits int) {
	e.normVolumeDigits = digits
}

func (e *TickGenerator) GetNext() (common.Tick, error) {
	var tick common.Tick

	if e.t >= e.steps {
		return tick, ErrEof
	}

	z := e.rng.NormFloat64()
	deltaLog := e.deltaLogPre1.Add(e.deltaLogPre2.Mul(fixed.FromFloat64(z)))
	e.lastPrice = e.lastPrice.Mul(deltaLog.Exp())

	e.updateSpread()

	tickInterval := e.generateTickInterval()
	e.lastTime = e.lastTime.Add(tickInterval)
	e.t++

	askVol, bidVol := e.generateVolumes()

	tick.Ask = e.lastPrice.Add(e.currentSpread)
	tick.Bid = e.lastPrice.Sub(e.currentSpread)
	tick.TimeStamp = e.lastTime
	tick.AskVolume = askVol
	tick.BidVolume = bidVol

	e.addTickNoise(&tick)

	tick.Ask = tick.Ask.Rescale(e.normPriceDigits)
	tick.Bid = tick.Bid.Rescale(e.normPriceDigits)

	tick.AskVolume = tick.AskVolume.Rescale(e.normVolumeDigits)
	tick.BidVolume = tick.BidVolume.Rescale(e.normVolumeDigits)

	tick.Source = tickGeneratorComponentName
	tick.Symbol = e.symbol
	tick.ExecutionId = utility.GetExecutionID()
	tick.TraceID = utility.CreateTraceID()

	return tick, nil
}

func (e *TickGenerator) updateSpread() {
	if e.spreadVolatility <= 0 {
		return
	}

	spreadChange := e.rng.NormFloat64() * e.spreadVolatility
	newSpread := e.currentSpread.Mul(fixed.FromFloat64(1.0 + spreadChange))

	if newSpread.Lt(e.minSpread) {
		e.currentSpread = e.minSpread
	} else if newSpread.Gt(e.maxSpread) {
		e.currentSpread = e.maxSpread
	} else {
		e.currentSpread = newSpread
	}
}

func (e *TickGenerator) generateTickInterval() time.Duration {
	if e.tickVariability <= 0 {
		return e.avgTickInterval
	}

	lambda := 1.0 / float64(e.avgTickInterval.Nanoseconds())
	interval := e.rng.ExpFloat64() / lambda

	minInterval := float64(e.avgTickInterval.Nanoseconds()) * (1.0 - e.tickVariability)
	maxInterval := float64(e.avgTickInterval.Nanoseconds()) * (1.0 + e.tickVariability*3)

	if interval < minInterval {
		interval = minInterval
	} else if interval > maxInterval {
		interval = maxInterval
	}

	return time.Duration(int64(interval))
}

func (e *TickGenerator) generateVolumes() (askVol, bidVol fixed.Point) {
	askVariation := e.rng.NormFloat64() * e.volumeVariance
	bidVariation := e.rng.NormFloat64() * e.volumeVariance

	askMultiplier := fixed.FromFloat64(1.0 + askVariation).Exp()
	bidMultiplier := fixed.FromFloat64(1.0 + bidVariation).Exp()

	askVol = e.avgVolume.Mul(askMultiplier)
	bidVol = e.avgVolume.Mul(bidMultiplier)

	// Ensure positive volumes
	if askVol.Lte(fixed.Zero) {
		askVol = fixed.One
	}
	if bidVol.Lte(fixed.Zero) {
		bidVol = fixed.One
	}

	return askVol, bidVol
}

func (e *TickGenerator) addTickNoise(tick *common.Tick) {
	tickSize := e.currentSpread.DivInt64(10)

	askNoise := fixed.FromFloat64(e.rng.NormFloat64() * 0.1).Mul(tickSize)
	bidNoise := fixed.FromFloat64(e.rng.NormFloat64() * 0.1).Mul(tickSize)

	tick.Ask = tick.Ask.Add(askNoise)
	tick.Bid = tick.Bid.Add(bidNoise)

	if tick.Bid.Gte(tick.Ask) {
		mid := tick.Bid.Add(tick.Ask).DivInt64(2)
		tick.Bid = mid.Sub(tickSize)
		tick.Ask = mid.Add(tickSize)
	}
}
