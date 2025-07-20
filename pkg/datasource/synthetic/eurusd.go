package synthetic

import (
	"log/slog"
	"math/rand"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func NewEURUSDTickGenerator(symbol string, rng *rand.Rand, duration time.Duration, mu, sigma float64) *TickGenerator {

	const (
		eurUsdStartPrice    = 1.0550
		eurUsdTypicalSpread = 0.00003 // 0.3 pips spread
		eurUsdMinSpread     = 0.00001 // 0.1 pips minimum
		eurUsdMaxSpread     = 0.00006 // 0.6 pips maximum

		avgTickIntervalSeconds = 1    // second average between ticks
		tickTimingVariability  = 0.45 // 45% timing variation

		avgVolumeUnits    = 1    // 1 units average volume
		volumeVariability = 0.65 // 65% volume variance

		spreadVolatility = 0.12 // 12% spread volatility

		normPriceDigits  = 5
		normVolumeDigits = 2
	)

	startTime := time.Now()

	totalSeconds := int64(duration.Seconds())
	avgTickInterval := time.Duration(avgTickIntervalSeconds * float64(time.Second))
	estimatedTicks := totalSeconds / int64(avgTickIntervalSeconds)

	secondsPerYear := 365.25 * 24 * 3600
	deltaT := fixed.FromFloat64(avgTickIntervalSeconds / secondsPerYear)

	startPrice := fixed.FromFloat64(eurUsdStartPrice)
	fullSpread := fixed.FromFloat64(eurUsdTypicalSpread)
	minSpread := fixed.FromFloat64(eurUsdMinSpread)
	maxSpread := fixed.FromFloat64(eurUsdMaxSpread)

	muFixed := fixed.FromFloat64(mu)
	sigmaFixed := fixed.FromFloat64(sigma)

	tickGenerator := NewTickGenerator(
		symbol,
		rng,
		startTime,
		startPrice,
		fullSpread,
		muFixed,
		sigmaFixed,
		deltaT,
		estimatedTicks,
	)

	tickGenerator.SetTickParameters(avgTickInterval, tickTimingVariability, fixed.FromInt(avgVolumeUnits, 0), volumeVariability)
	tickGenerator.SetSpreadDynamics(spreadVolatility, minSpread, maxSpread)
	tickGenerator.SetPriceDigits(normPriceDigits)
	tickGenerator.SetVolumeDigits(normVolumeDigits)

	slog.Debug("EURUSD synthetic tick generator configuration",
		"duration", duration,
		"mu_annual", mu,
		"sigma_annual", sigma,
		"start_price", eurUsdStartPrice,
		"avg_spread_pips", eurUsdTypicalSpread*100000,
		"avg_tick_interval_sec", avgTickIntervalSeconds,
		"estimated_ticks", estimatedTicks,
		"start_time", startTime,
	)

	return tickGenerator
}
