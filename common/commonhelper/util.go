package commonhelper

func Addr[T any](v T) *T { return &v }

func DefaultString(item *string) string {
	if item == nil {
		return "<nil>"
	}
	return *item
}
