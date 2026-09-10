//go:build !unix

package terminal

import "time"

// waitReadable is unavailable off unix, so width probing is skipped and the
// built-in width tables are kept.
func waitReadable(fd int, deadline time.Time) (bool, error) {
	return false, ErrProbeUnsupported
}
