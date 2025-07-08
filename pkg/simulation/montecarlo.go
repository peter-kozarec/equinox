package simulation

import (
	"log/slog"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"

	"math/rand"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	pointFive = fixed.FromInt64(5, 1)
)

type MonteCarloExecutor struct {
	simulator *Simulator
	rng       *rand.Rand

	startTime  time.Time
	startPrice fixed.Point
	baseSpread fixed.Point
	mu         fixed.Point
	sigma      fixed.Point
	deltaT     fixed.Point
	steps      int64
	t          int64

	// Tick timing parameters
	avgTickInterval time.Duration
	tickVariability float64

	// Volume parameters
	avgVolume      fixed.Point
	volumeVariance float64

	// Price model parameters
	deltaLogPre1 fixed.Point
	deltaLogPre2 fixed.Point

	// Spread dynamics
	spreadVolatility float64
	minSpread        fixed.Point
	maxSpread        fixed.Point

	lastTime      time.Time
	lastPrice     fixed.Point
	currentSpread fixed.Point

	tick common.Tick

	normPriceDigits  int
	normVolumeDigits int
}

func NewMonteCarloExecutor(
	sim *Simulator,
	rng *rand.Rand,
	startTime time.Time,
	startPrice, fullSpread, mu, sigma, deltaT fixed.Point,
	steps int64) *MonteCarloExecutor {

	avgTickInterval := time.Duration(333_000_000) // ~333ms default

	return &MonteCarloExecutor{
		simulator: sim,
		rng:       rng,

		startTime:  startTime,
		startPrice: startPrice,
		baseSpread: fullSpread.DivInt64(2), // Half spread for bid/ask calculation
		mu:         mu,
		sigma:      sigma,
		deltaT:     deltaT,
		steps:      steps,

		// Tick timing - realistic microsecond-level variations
		avgTickInterval: avgTickInterval,
		tickVariability: 0.3, // 30% variation in tick timing

		// Volume parameters - more realistic volume simulation
		avgVolume:      fixed.FromInt64(100, 0), // Average 100 units
		volumeVariance: 0.5,                     // 50% variance in volume

		// Spread dynamics
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
func NewEurUsdMonteCarloTickSimulator(
	simulator *Simulator,
	rng *rand.Rand,
	duration time.Duration,
	mu, sigma float64) *MonteCarloExecutor {

	// EURUSD-specific configuration
	const (
		// Market characteristics
		eurUsdStartPrice    = 1.0550  // Typical EURUSD starting price
		eurUsdTypicalSpread = 0.00003 // 0.3 pips spread
		eurUsdMinSpread     = 0.00001 // 0.1 pips minimum
		eurUsdMaxSpread     = 0.00006 // 0.6 pips maximum

		// Tick timing (realistic for EURUSD)
		avgTickIntervalSeconds = 1    // second average between ticks
		tickTimingVariability  = 0.45 // 45% timing variation

		// Volume characteristics
		avgVolumeUnits    = 1    // 1 units average volume
		volumeVariability = 0.65 // 65% volume variance

		// Spread dynamics
		spreadVolatility = 0.12 // 12% spread volatility

		// Normalization digits
		normPriceDigits  = 5
		normVolumeDigits = 2
	)

	// Setup timing
	startTime := time.Now()

	// Convert duration to number of ticks
	totalSeconds := int64(duration.Seconds())
	avgTickInterval := time.Duration(avgTickIntervalSeconds * float64(time.Second))
	estimatedTicks := totalSeconds / int64(avgTickIntervalSeconds)

	// Time delta for price model (convert to fraction of year)
	secondsPerYear := 365.25 * 24 * 3600
	deltaT := fixed.FromFloat64(avgTickIntervalSeconds / secondsPerYear)

	// Convert price and spread to fixed point
	startPrice := fixed.FromFloat64(eurUsdStartPrice)
	fullSpread := fixed.FromFloat64(eurUsdTypicalSpread)
	minSpread := fixed.FromFloat64(eurUsdMinSpread)
	maxSpread := fixed.FromFloat64(eurUsdMaxSpread)

	// Convert mu and sigma to fixed point
	muFixed := fixed.FromFloat64(mu)
	sigmaFixed := fixed.FromFloat64(sigma)

	// Create the base Monte Carlo executor
	executor := NewMonteCarloExecutor(
		simulator,
		rng,
		startTime,
		startPrice,
		fullSpread,
		muFixed,
		sigmaFixed,
		deltaT,
		estimatedTicks,
	)

	// Configure EURUSD-specific tick parameters
	executor.SetTickParameters(
		avgTickInterval,
		tickTimingVariability,
		fixed.FromInt64(int64(avgVolumeUnits), 0),
		volumeVariability,
	)

	// Configure EURUSD-specific spread dynamics
	executor.SetSpreadDynamics(
		spreadVolatility,
		minSpread,
		maxSpread,
	)

	executor.normPriceDigits = normPriceDigits
	executor.normVolumeDigits = normVolumeDigits

	// Log configuration
	slog.Debug("EURUSD Monte Carlo Tick Simulator configured",
		"duration", duration,
		"mu_annual", mu,
		"sigma_annual", sigma,
		"start_price", eurUsdStartPrice,
		"avg_spread_pips", eurUsdTypicalSpread*100000,
		"avg_tick_interval_sec", avgTickIntervalSeconds,
		"estimated_ticks", estimatedTicks,
		"start_time", startTime,
	)

	return executor
}

// SetTickParameters allows customization of tick characteristics
func (e *MonteCarloExecutor) SetTickParameters(
	avgInterval time.Duration,
	intervalVariability float64,
	avgVol fixed.Point,
	volVariance float64) {

	e.avgTickInterval = avgInterval
	e.tickVariability = intervalVariability
	e.avgVolume = avgVol
	e.volumeVariance = volVariance
}

// SetSpreadDynamics configures dynamic spread behavior
func (e *MonteCarloExecutor) SetSpreadDynamics(
	volatility float64,
	minSpread, maxSpread fixed.Point) {

	e.spreadVolatility = volatility
	e.minSpread = minSpread
	e.maxSpread = maxSpread
}

func (e *MonteCarloExecutor) DoOnce() error {
	if e.t >= e.steps {
		return mapper.ErrEof
	}

	// Generate next price using Geometric Brownian Motion
	z := e.rng.NormFloat64()
	deltaLog := e.deltaLogPre1.Add(e.deltaLogPre2.Mul(fixed.FromFloat64(z)))
	e.lastPrice = e.lastPrice.Mul(deltaLog.Exp())

	// Dynamic spread based on volatility
	e.updateSpread()

	// Variable tick timing - more realistic than fixed intervals
	tickInterval := e.generateTickInterval()
	e.lastTime = e.lastTime.Add(tickInterval)
	e.t++

	// Generate realistic volumes
	askVol, bidVol := e.generateVolumes()

	// Build tick with realistic bid/ask spread
	e.tick.Ask = e.lastPrice.Add(e.currentSpread)
	e.tick.Bid = e.lastPrice.Sub(e.currentSpread)
	e.tick.TimeStamp = e.lastTime.UnixNano()
	e.tick.AskVolume = askVol
	e.tick.BidVolume = bidVol

	// Optional: Add some tick-level noise for more realism
	e.addTickNoise()

	e.tick.Ask = e.tick.Ask.Rescale(e.normPriceDigits)
	e.tick.Bid = e.tick.Bid.Rescale(e.normPriceDigits)

	e.tick.AskVolume = e.tick.AskVolume.Rescale(e.normVolumeDigits)
	e.tick.BidVolume = e.tick.BidVolume.Rescale(e.normVolumeDigits)

	if err := e.simulator.OnTick(e.tick); err != nil {
		return err
	}

	return nil
}

// updateSpread simulates dynamic spread changes based on market conditions
func (e *MonteCarloExecutor) updateSpread() {
	if e.spreadVolatility <= 0 {
		return
	}

	// Spread tends to widen during high volatility
	spreadChange := e.rng.NormFloat64() * e.spreadVolatility
	newSpread := e.currentSpread.Mul(fixed.FromFloat64(1.0 + spreadChange))

	// Clamp spread within bounds
	if newSpread.Lt(e.minSpread) {
		e.currentSpread = e.minSpread
	} else if newSpread.Gt(e.maxSpread) {
		e.currentSpread = e.maxSpread
	} else {
		e.currentSpread = newSpread
	}
}

// generateTickInterval creates realistic variable tick timing
func (e *MonteCarloExecutor) generateTickInterval() time.Duration {
	if e.tickVariability <= 0 {
		return e.avgTickInterval
	}

	// Use exponential distribution for more realistic tick intervals
	// This creates clustering and gaps similar to real market data
	lambda := 1.0 / float64(e.avgTickInterval.Nanoseconds())
	interval := e.rng.ExpFloat64() / lambda

	// Add some bounds to prevent extreme values
	minInterval := float64(e.avgTickInterval.Nanoseconds()) * (1.0 - e.tickVariability)
	maxInterval := float64(e.avgTickInterval.Nanoseconds()) * (1.0 + e.tickVariability*3)

	if interval < minInterval {
		interval = minInterval
	} else if interval > maxInterval {
		interval = maxInterval
	}

	return time.Duration(int64(interval))
}

// generateVolumes creates realistic bid/ask volumes
func (e *MonteCarloExecutor) generateVolumes() (askVol, bidVol fixed.Point) {
	// Generate volumes with log-normal distribution for realism
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

// addTickNoise adds small random variations to simulate market microstructure
func (e *MonteCarloExecutor) addTickNoise() {
	// Small random adjustments to bid/ask to simulate order book dynamics
	tickSize := e.currentSpread.DivInt64(10) // Minimum tick size

	askNoise := fixed.FromFloat64(e.rng.NormFloat64() * 0.1).Mul(tickSize)
	bidNoise := fixed.FromFloat64(e.rng.NormFloat64() * 0.1).Mul(tickSize)

	e.tick.Ask = e.tick.Ask.Add(askNoise)
	e.tick.Bid = e.tick.Bid.Add(bidNoise)

	// Ensure bid < ask
	if e.tick.Bid.Gte(e.tick.Ask) {
		mid := e.tick.Bid.Add(e.tick.Ask).DivInt64(2)
		e.tick.Bid = mid.Sub(tickSize)
		e.tick.Ask = mid.Add(tickSize)
	}
}
