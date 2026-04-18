package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func HashParts(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:])
}
