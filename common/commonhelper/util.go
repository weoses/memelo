package commonhelper

func Addr[T any](v T) *T { return &v }
