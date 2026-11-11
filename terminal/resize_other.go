//go:build !unix

package terminal

import "time"

// resizePollInterval is how often the terminal size is read where no signal
// reports a change.
const resizePollInterval = 100 * time.Millisecond

// notifyResize returns a channel that receives every resizePollInterval, and
// a func that stops it. Windows has no SIGWINCH: its console reports a resize
// among the input records that reading stdin as a file skips.
func notifyResize() (<-chan time.Time, func()) {
	t := time.NewTicker(resizePollInterval)
	return t.C, t.Stop
}
