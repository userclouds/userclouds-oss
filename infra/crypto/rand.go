package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

// MustRandomBase64 creates a cryptographically secure n-byte, base64-encoded string.
// Suitable for use as a token, nonce, etc.
// Note: base64 takes 4 bytes to encode 3 source bytes, so the string length
// will be ceil[n/3] * 4.
// Will panic if unable to generate a random string.
func MustRandomBase64(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// If this fails it's not likely recoverable.
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// MustRandomHex creates a cryptographically secure n-byte, hex-encoded string.
// Suitable for use as a token, nonce, etc.
// Note: hex takes 2 bytes to encode 1 source byte, so the string length
// will be (n * 2).
// Will panic if unable to generate a random string.
func MustRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// If this fails it's not likely recoverable.
		panic(err)
	}
	return hex.EncodeToString(b)
}

// MustRandomDigits creates a cryptographically secure n-character string of [0-9]
func MustRandomDigits(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// If this fails it's not likely recoverable.
		panic(err)
	}
	str := ""
	for _, c := range b {
		// TODO: this has a slight bias (~4%) in favor of the digits [0,5]; should probably discard
		// random byte values 250+, or generate random int32/int64 values so the bias is negligible.
		str = str + string('0'+(c%10))
	}
	return str
}
