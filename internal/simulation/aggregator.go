package simulation

import (
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
	"time"
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

func (aggregator *Aggregator) OnTick(tick *model.Tick) error {
	ts := time.Unix(0, tick.TimeStamp)
	barTS := ts.Truncate(aggregator.interval).UnixNano()
	price := (tick.Bid + tick.Ask) / 2
	volume := tick.BidVolume + tick.AskVolume

	// Gap detection — flush and reset
	if aggregator.currentBar != nil && barTS != aggregator.currentBar.TimeStamp {
		if err := aggregator.router.Post(bus.BarEvent, aggregator.currentBar); err != nil {
			return err
		}
		aggregator.currentBar = nil
	}

	if aggregator.currentBar == nil {
		aggregator.currentBar = &model.Bar{
			TimeStamp: barTS,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    volume,
			Period:    aggregator.interval,
		}
	} else {
		if price > aggregator.currentBar.High {
			aggregator.currentBar.High = price
		}
		if price < aggregator.currentBar.Low {
			aggregator.currentBar.Low = price
		}
		aggregator.currentBar.Close = price
		aggregator.currentBar.Volume += volume
	}

	aggregator.lastTS = tick.TimeStamp
	return nil
}

func (aggregator *Aggregator) Flush() error {
	if aggregator.currentBar != nil {
		err := aggregator.router.Post(bus.BarEvent, aggregator.currentBar)
		aggregator.currentBar = nil
		return err
	}
	return nil
}
