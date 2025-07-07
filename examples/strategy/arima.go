package strategy

import (
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/models/arima"
	"go.uber.org/zap"
)

// ArimaAdvisor is a test strategy, not meant for production
type ArimaAdvisor struct {
	logger *zap.Logger
	router *bus.Router

	model *arima.Model
}

func NewArimaAdvisor(logger *zap.Logger, router *bus.Router, model *arima.Model) *ArimaAdvisor {
	return &ArimaAdvisor{
		logger: logger,
		router: router,
		model:  model,
	}
}

func (a *ArimaAdvisor) OnNewBar(b common.Bar) {
	a.model.AddPoint(b.Close)

	if a.model.ShouldReestimate() {
		if err := a.model.Estimate(); err != nil {
			a.logger.Warn("failed to estimate arima model", zap.Error(err))
			return
		}
	}

	if !a.model.IsEstimated() {
		return
	}

	if err := a.model.ValidateModel(); err != nil {
		a.logger.Warn("validation failed, reestimating arima model", zap.Error(err))
		if err := a.model.Estimate(); err != nil {
			a.logger.Warn("failed to re-estimate arima model", zap.Error(err))
			return
		}
		if err := a.model.ValidateModel(); err != nil {
			a.logger.Warn("validation failed after re-estimation, aborting", zap.Error(err))
			return
		}
	}

	forecast, err := a.model.Forecast(1)
	if err != nil || forecast == nil || len(forecast) != 1 {
		a.logger.Warn("failed to forecast arima model", zap.Error(err))
		return
	}

	a.logger.Info("arima model estimated",
		zap.String("current_close", b.Close.String()),
		zap.String("next_forecasted_close", forecast[0].PointForecast.Rescale(5).String()))
}
