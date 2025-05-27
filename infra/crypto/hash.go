package crypto

import (
	"crypto/md5"
	"encoding/hex"
)

// GetMD5Hash returns the md5 hash of the given string
func GetMD5Hash(secret string) string {
	secretHash := md5.Sum([]byte(secret))
	return hex.EncodeToString(secretHash[:])
}
