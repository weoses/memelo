package commonhelper

func Addr[T any](v T) *T { return &v }

func DefaultString(item *string) string {
	if item == nil {
		return "<nil>"
	}
	return *item
}

func TransformSlice[F any, T any](from []F, to []T, transformer func(F) T) []T {
	for _, f := range from {
		to = append(to, transformer(f))
	}
	return to
}
