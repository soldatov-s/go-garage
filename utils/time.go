package utils

import (
	"time"
)

// UnixTimeToUTZ convert unix time to local time.Time
// If unixTime equal zero return time.Now()
func UnixTimeToUTZ(unixTime int) time.Time {
	if unixTime == 0 {
		return time.Now()
	}
	return time.Unix(int64(unixTime), 0)
}

// UnixTimeToUTC convert unix time to UTC time.Time
// If unixTime equal zero return time.Now().UTC
func UnixTimeToUTC(unixTime int) time.Time {
	if unixTime == 0 {
		return time.Now().UTC()
	}
	return time.Unix(int64(unixTime), 0).UTC()
}
