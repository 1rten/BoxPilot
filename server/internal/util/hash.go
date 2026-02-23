package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// SHA256Hex returns SHA256 hash of data as hex string.
func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// JSONHash returns SHA256 hex of JSON-marshalled v (or empty string on error).
func JSONHash(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return SHA256Hex(b)
}
