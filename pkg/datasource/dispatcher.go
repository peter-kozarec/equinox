package datasource

import (
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
)

type TickDataSource interface {
	GetNext() (common.Tick, error)
}

func CreateTickDispatcher(r *bus.Router, ds TickDataSource) func() error {
	return func() error {
		var tick common.Tick
		var err error

		if tick, err = ds.GetNext(); err != nil {
			return err
		}
		if err = r.Post(bus.TickEvent, tick); err != nil {
			return err
		}
		return nil
	}
}
