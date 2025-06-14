package middleware

func Chain[T any](wrappers ...func(T) T) func(T) T {
	return func(handler T) T {
		for i := len(wrappers) - 1; i >= 0; i-- {
			handler = wrappers[i](handler)
		}
		return handler
	}
}
