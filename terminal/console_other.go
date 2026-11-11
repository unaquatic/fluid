//go:build !windows

package terminal

import "os"

// enableVirtualTerminal returns a nil restore func, as terminals outside
// Windows interpret escape sequences without a mode to enable.
func enableVirtualTerminal(*os.File) (restore func() error, err error) {
	return nil, nil
}

// openConsoleInput returns nil, as a terminal outside Windows sends its
// input as bytes, which are read from f itself.
func openConsoleInput(*os.File) (consoleInput, error) {
	return nil, nil
}
