//go:build !unix && !windows

package terminal

import "time"

// resizePollInterval is how often the terminal size is read where nothing
// reports a change.
const resizePollInterval = 100 * time.Millisecond

// resizeNotice is what tells that the terminal may have changed size: a
// tick of a ticker.
type resizeNotice = time.Time

// notifyResize returns a channel that receives every resizePollInterval,
// and a func that stops it, as no signal reports a resize here.
func (t *Backend) notifyResize() (<-chan resizeNotice, func()) {
	tk := time.NewTicker(resizePollInterval)
	return tk.C, tk.Stop
}
