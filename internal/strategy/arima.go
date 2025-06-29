package strategy

import "github.com/peter-kozarec/equinox/pkg/models/arima"

type ArimaAdvisor struct {
	model *arima.Model
}

func NewArimaAdvisor(model *arima.Model) *ArimaAdvisor {
	return &ArimaAdvisor{
		model: model,
	}
}
