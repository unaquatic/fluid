//go:build unix

package terminal

import (
	"errors"
	"time"

	"golang.org/x/sys/unix"
)

// waitReadable reports whether fd has data to read before deadline.
//
// This uses poll(2) rather than os.File.SetReadDeadline because a terminal
// stdin is opened in blocking mode and is therefore not registered with the
// runtime's network poller; SetReadDeadline returns ErrNoDeadline for it. A
// bare Read would then block forever when the terminal ignores the request.
func waitReadable(fd int, deadline time.Time) (bool, error) {
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		ms := int(time.Until(deadline).Milliseconds())
		if ms <= 0 {
			return false, nil
		}
		n, err := unix.Poll(fds, ms)
		if errors.Is(err, unix.EINTR) {
			continue // a signal interrupted the wait; the deadline still governs
		}
		if err != nil {
			return false, err
		}
		return n > 0, nil
	}
}
