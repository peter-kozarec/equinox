package arima

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type ModelDiagnostics struct {
	LogLikelihood   fixed.Point
	AIC             fixed.Point
	BIC             fixed.Point
	AICC            fixed.Point
	RMSE            fixed.Point
	MAE             fixed.Point
	MAPE            fixed.Point
	LjungBoxPValue  fixed.Point
	JarqueBeraTest  fixed.Point
	IsStationary    bool
	ConvergenceCode int
	Iterations      int
}
