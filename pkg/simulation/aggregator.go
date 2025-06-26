package simulation

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/model"
)

type Aggregator struct {
	interval   time.Duration
	router     *bus.Router
	currentBar *model.Bar
	lastTS     int64
}

func NewAggregator(interval time.Duration, bus *bus.Router) *Aggregator {
	return &Aggregator{
		interval: interval,
		router:   bus,
	}
}

func (a *Aggregator) OnTick(tick model.Tick) error {
	ts := time.Unix(0, tick.TimeStamp)
	barTS := ts.Truncate(a.interval).UnixNano()
	price := tick.Average()
	volume := tick.AggregatedVolume()

	// Gap detection — flush and reset
	if a.currentBar != nil && barTS != a.currentBar.TimeStamp {
		if err := a.router.Post(bus.BarEvent, *a.currentBar); err != nil {
			return err
		}
		a.currentBar = nil
	}

	if a.currentBar == nil {
		a.currentBar = &model.Bar{
			TimeStamp: barTS,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    volume,
			Period:    a.interval,
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
	return nil
}

func (a *Aggregator) Flush() error {
	if a.currentBar != nil && a.currentBar.TimeStamp != 0 {
		err := a.router.Post(bus.BarEvent, *a.currentBar)
		a.currentBar = nil
		return err
	}
	return nil
}
