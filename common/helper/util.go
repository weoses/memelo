package helper

import (
	"crypto/md5"
	"encoding/hex"
)

func Addr[T any](v T) *T { return &v }

func DefaultString(item *string) string {
	if item == nil {
		return "<nil>"
	}
	return *item
}

func TransformSlice[F any, T any](from []F, to []T, transformer func(F) T) []T {
	for i, f := range from {
		to[i] = transformer(f)
	}
	return to
}

func CalcHash(base64Image string) string {
	hasher := md5.New()
	hasher.Write([]byte(base64Image))
	byteHash := hasher.Sum(nil)
	return hex.EncodeToString(byteHash)
}
