package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a random hex string made from n random bytes (2n characters).
func New(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
