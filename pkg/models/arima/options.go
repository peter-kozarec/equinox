package arima

type ModelOption func(*Model)

func WithEstimationMethod(method EstimationMethod) ModelOption {
	return func(m *Model) {
		m.method = method
	}
}

func WithConstant(include bool) ModelOption {
	return func(m *Model) {
		m.includeConstant = include
	}
}

func WithSeasonal(period int) ModelOption {
	return func(m *Model) {
		if period > 1 {
			m.seasonal = true
			m.seasonalPeriod = period
		}
	}
}
