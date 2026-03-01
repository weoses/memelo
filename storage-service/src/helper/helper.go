package helper

import (
	"crypto/md5"
	"encoding/hex"
)

func CalcHash(base64Image string) string {
	hasher := md5.New()
	hasher.Write([]byte(base64Image))
	byteHash := hasher.Sum(nil)
	return hex.EncodeToString(byteHash)
}
