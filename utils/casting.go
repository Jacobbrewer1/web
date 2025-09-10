package utils

func CastWithDefault[T any](value any, defaultValue T) T {
	if v, ok := value.(T); ok {
		return v
	}
	return defaultValue
}
