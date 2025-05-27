package column

import (
	"time"
)

// dateToString formats a date as a time.DateOnly string
func dateToString(d time.Time) string {
	return d.Format(time.DateOnly)
}

// datesToStrings formats a collection of dates as time.DateOnly strings
func datesToStrings(dates []time.Time) []string {
	strings := make([]string, len(dates))
	for i, d := range dates {
		strings[i] = dateToString(d)
	}
	return strings
}

// timeToString formats a time as a time.RFC3339 string
func timeToString(t time.Time) string {
	return t.Format(time.RFC3339)
}

// timesToStrings formats a collection of times as time.RFC3339 strings
func timesToStrings(times []time.Time) []string {
	strings := make([]string, len(times))
	for i, t := range times {
		strings[i] = timeToString(t)
	}
	return strings
}
