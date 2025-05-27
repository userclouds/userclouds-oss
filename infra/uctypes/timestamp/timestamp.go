package timestamp

import "time"

// Normalize sets location to UTC and rounds the timestamp to microsecond
// precision, since psql has microsecond precision but go may have nano
// or microsecond precision based on platform
func Normalize(timestamp time.Time) time.Time {
	return timestamp.UTC().Round(time.Microsecond)
}

// NormalizedEqual returns true if the normalized reprentations of both
// timestamps are equal
func NormalizedEqual(left time.Time, right time.Time) bool {
	return Normalize(left) == Normalize(right)
}
