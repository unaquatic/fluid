//go:build !unix

package terminal

import "time"

// waitReadable reports unsupported outside Unix so callers retain built-in
// width tables.
func waitReadable(fd int, deadline time.Time) (bool, error) {
	return false, ErrProbeUnsupported
}
