package domain

import (
	"crypto/rand"
	"encoding/hex"
)

const RFC3339Millis = "2006-01-02T15:04:05.000Z07:00"

func GenerateRandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
