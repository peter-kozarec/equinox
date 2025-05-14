package simulation

import "peter-kozarec/equinox/internal/model"

type Simulator struct{}

func (simulator *Simulator) OnTick(tick *model.Tick) error {
	return nil
}
