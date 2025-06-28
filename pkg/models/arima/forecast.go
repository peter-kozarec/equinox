package arima

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type ForecastResult struct {
	PointForecast      fixed.Point
	StandardError      fixed.Point
	ConfidenceInterval struct {
		Lower95 fixed.Point
		Upper95 fixed.Point
		Lower80 fixed.Point
		Upper80 fixed.Point
	}
	PredictionInterval struct {
		Lower95 fixed.Point
		Upper95 fixed.Point
		Lower80 fixed.Point
		Upper80 fixed.Point
	}
}
