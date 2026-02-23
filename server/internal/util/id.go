package util

import "github.com/google/uuid"

// NewID returns a new UUID string.
func NewID() string {
	return uuid.New().String()
}
