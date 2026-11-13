package fixtures

func setData[T any](def T, value *T) T {
	if value == nil {
		return def
	}
	return *value
}
