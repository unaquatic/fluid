//go:build unix

package terminal

import (
	"os"
	"os/signal"
	"syscall"
)

// resizeNotice is what tells that the terminal may have changed size: a
// SIGWINCH.
type resizeNotice = os.Signal

// notifyResize returns a channel that receives when the terminal may have
// changed size, on SIGWINCH, and a func that stops it.
func (t *Backend) notifyResize() (<-chan resizeNotice, func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	return c, func() { signal.Stop(c) }
}
