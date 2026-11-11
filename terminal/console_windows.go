//go:build windows

package terminal

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// enableVirtualTerminal has the Windows console that f writes to interpret
// ANSI and xterm escape sequences, and defer wrapping at the last column and
// leave the cursor column alone on a line feed, as a terminal in raw mode
// does. It returns a func that restores the console's mode, or nil when f is
// not a console, such as a pipe or a terminal that is not a Windows console,
// which needs no change.
//
// It fails when the console cannot interpret escape sequences, as before
// Windows 10.
func enableVirtualTerminal(f *os.File) (restore func() error, err error) {
	h := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return nil, nil
	}
	vt := mode | windows.ENABLE_PROCESSED_OUTPUT | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	// Consoles before Windows 10 1703 reject DISABLE_NEWLINE_AUTO_RETURN,
	// but may still interpret escape sequences without it.
	if err := windows.SetConsoleMode(h, vt|windows.DISABLE_NEWLINE_AUTO_RETURN); err != nil {
		if err := windows.SetConsoleMode(h, vt); err != nil {
			return nil, fmt.Errorf("terminal: console does not support virtual terminal sequences: %w", err)
		}
	}
	return func() error { return windows.SetConsoleMode(h, mode) }, nil
}
