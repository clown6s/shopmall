package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashSHA256 returns the SHA-256 hex digest of s.
func HashSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// DerefStr safely dereferences a *string, returning "" if nil.
func DerefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
