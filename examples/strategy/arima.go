package strategy

import (
	"log/slog"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/models/arima"
)

// ArimaAdvisor is a test strategy, not meant for production
type ArimaAdvisor struct {
	router *bus.Router

	model *arima.Model
}

func NewArimaAdvisor(router *bus.Router, model *arima.Model) *ArimaAdvisor {
	return &ArimaAdvisor{
		router: router,
		model:  model,
	}
}

func (a *ArimaAdvisor) OnNewBar(b common.Bar) {
	a.model.AddPoint(b.Close)

	if a.model.ShouldReestimate() {
		if err := a.model.Estimate(); err != nil {
			slog.Warn("failed to estimate arima model", "error", err)
			return
		}
	}

	if !a.model.IsEstimated() {
		return
	}

	if err := a.model.ValidateModel(); err != nil {
		slog.Warn("validation failed, reestimating arima model", "error", err)
		if err := a.model.Estimate(); err != nil {
			slog.Warn("failed to re-estimate arima model", "error", err)
			return
		}
		if err := a.model.ValidateModel(); err != nil {
			slog.Warn("validation failed after re-estimation, aborting", "error", err)
			return
		}
	}

	forecast, err := a.model.Forecast(1)
	if err != nil || forecast == nil || len(forecast) != 1 {
		slog.Warn("failed to forecast arima model", "error", err)
		return
	}

	slog.Info("arima model estimated",
		"current_close", b.Close,
		"next_forecasted_close", forecast[0].PointForecast.Rescale(5))
}
