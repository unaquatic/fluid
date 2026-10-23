//go:build unix

package terminal

import (
	"errors"
	"time"

	"golang.org/x/sys/unix"
)

// waitReadable reports whether fd has data to read before deadline.
//
// Terminal input does not support os.File.SetReadDeadline, so poll bounds the
// wait without changing the file's blocking mode.
func waitReadable(fd int, deadline time.Time) (bool, error) {
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		ms := int(time.Until(deadline).Milliseconds())
		if ms <= 0 {
			return false, nil
		}
		n, err := unix.Poll(fds, ms)
		if errors.Is(err, unix.EINTR) {
			continue // Retry an interrupted wait under the original deadline.
		}
		if err != nil {
			return false, err
		}
		return n > 0, nil
	}
}
