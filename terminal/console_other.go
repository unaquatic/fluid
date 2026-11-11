//go:build !windows

package terminal

import "os"

// enableVirtualTerminal returns a nil restore func, as terminals outside
// Windows interpret escape sequences without a mode to enable.
func enableVirtualTerminal(*os.File) (restore func() error, err error) {
	return nil, nil
}
