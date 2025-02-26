package hash

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5(token string) string {
	hash := md5.New()
	hash.Write([]byte(token))

	return hex.EncodeToString(hash.Sum(nil))
}
