package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
)

func MD5(token string) string {
	hash := md5.New()
	hash.Write([]byte(token))

	return hex.EncodeToString(hash.Sum(nil))
}

func Fields(fields []string) string {
	data := ""
	for _, field := range fields {
		data += field
	}

	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}
