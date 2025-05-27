package timer

import "time"

// Timer is a simple utility that allows a caller to capture
// elapsed time. The timer starts when created and can be
// reset via the Reset method.
type Timer struct {
	startTime time.Time
}

// Start will create a new Timer
func Start() Timer {
	return Timer{
		startTime: time.Now().UTC(),
	}
}

// Elapsed will return the elapsed duration since the last reset
func (t Timer) Elapsed() time.Duration {
	return time.Now().UTC().Sub(t.startTime)
}

// Reset will reset the timer, returning the elapsed
// duration since the last reset
func (t *Timer) Reset() time.Duration {
	elapsed := t.Elapsed()
	t.startTime = time.Now().UTC()
	return elapsed
}
