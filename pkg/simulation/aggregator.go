package simulation

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
)

const (
	componentName = "simulation"
)

type Aggregator struct {
	interval   time.Duration
	instrument common.Instrument
	router     *bus.Router
	currentBar *common.Bar
	lastTS     time.Time
}

func NewAggregator(interval time.Duration, instrument common.Instrument, bus *bus.Router) *Aggregator {
	return &Aggregator{
		interval:   interval,
		instrument: instrument,
		router:     bus,
	}
}

func (a *Aggregator) OnTick(tick common.Tick) {
	ts := tick.TimeStamp
	barTS := ts.Truncate(a.interval)
	price := tick.Bid
	volume := tick.AskVolume.Add(tick.BidVolume)

	// Gap detection — flush and reset
	if a.currentBar != nil && barTS.UnixNano() != a.currentBar.TimeStamp.UnixNano() {
		if err := a.router.Post(bus.BarEvent, *a.currentBar); err != nil {
			slog.Warn("unable to post bar", "error", err)
		}
		a.currentBar = nil
	}

	if a.currentBar == nil {
		a.currentBar = &common.Bar{
			Source:      componentName,
			Symbol:      a.instrument.Symbol,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   barTS,
			Open:        price,
			High:        price,
			Low:         price,
			Close:       price,
			Volume:      volume,
			Period:      a.interval,
		}
	} else {
		if price.Gt(a.currentBar.High) {
			a.currentBar.High = price
		}
		if price.Lt(a.currentBar.Low) {
			a.currentBar.Low = price
		}
		a.currentBar.Close = price
		a.currentBar.Volume = a.currentBar.Volume.Add(volume)
	}

	a.lastTS = tick.TimeStamp
}

func (a *Aggregator) Flush() error {
	if a.currentBar != nil && !a.currentBar.TimeStamp.IsZero() {
		err := a.router.Post(bus.BarEvent, *a.currentBar)
		a.currentBar = nil
		return err
	}
	return nil
}
