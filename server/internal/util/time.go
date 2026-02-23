package util

import "time"

const RFC3339 = "2006-01-02T15:04:05Z07:00"

// NowRFC3339 returns current time in RFC3339.
func NowRFC3339() string {
	return time.Now().UTC().Format(RFC3339)
}
