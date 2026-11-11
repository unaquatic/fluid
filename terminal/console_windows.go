//go:build windows

package terminal

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// procReadConsoleInput is ReadConsoleInputW, which golang.org/x/sys/windows
// leaves out.
var procReadConsoleInput = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReadConsoleInputW")

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

// openConsoleInput returns a reader of the input records of the Windows
// console that f reads from, which f's raw mode must have put in virtual
// terminal input mode, or nil when f is not a console. It has the console
// report its resizes among its input, until f's mode is restored.
//
// Reading the records, rather than f, passes Ctrl-Z on, which reading a
// console as a file takes for its end, and receives the resizes, which
// reading it as a file skips.
func openConsoleInput(f *os.File) (consoleInput, error) {
	h := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return nil, nil
	}
	if err := windows.SetConsoleMode(h, mode|windows.ENABLE_WINDOW_INPUT); err != nil {
		return nil, fmt.Errorf("terminal: failed to have the console report resizes: %w", err)
	}
	c, err := newConsoleReader(h)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// consoleReader reads the input records of a Windows console, decoded by a
// ConsoleDecoder, until it is closed.
type consoleReader struct {
	h windows.Handle
	// stop is an event that Close signals, which ends a Read waiting for
	// input.
	stop    windows.Handle
	closing atomic.Bool
	resized chan struct{}

	mu      sync.Mutex // Held by Read, which Close waits for to close stop.
	dec     ConsoleDecoder
	records [64]ConsoleRecord
	pending []byte // Decoded input that Read has yet to return.
	off     int    // How much of pending Read has returned.
}

// newConsoleReader returns a reader of the input records of the console
// input handle h.
func newConsoleReader(h windows.Handle) (*consoleReader, error) {
	stop, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("terminal: failed to create the console input's stop event: %w", err)
	}
	return &consoleReader{h: h, stop: stop, resized: make(chan struct{}, 1)}, nil
}

// Read waits for input records, and reads the bytes that they stand for,
// as ConsoleDecoder decodes them. It fails with os.ErrClosed once c is
// closed.
func (c *consoleReader) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.off == len(c.pending) {
		if c.closing.Load() {
			return 0, os.ErrClosed
		}
		// stop comes first, so that it wins over input waiting too.
		ev, err := windows.WaitForMultipleObjects([]windows.Handle{c.stop, c.h}, false, windows.INFINITE)
		if err != nil {
			return 0, err
		}
		if ev == windows.WAIT_OBJECT_0 {
			return 0, os.ErrClosed
		}
		var n uint32
		r, _, err := procReadConsoleInput.Call(uintptr(c.h), uintptr(unsafe.Pointer(&c.records[0])), uintptr(len(c.records)), uintptr(unsafe.Pointer(&n)))
		if r == 0 {
			return 0, err
		}
		var resized bool
		c.pending, resized = c.dec.Decode(c.pending[:0], c.records[:n])
		c.off = 0
		if resized {
			select {
			case c.resized <- struct{}{}:
			default: // A resize is reported already.
			}
		}
	}
	n := copy(b, c.pending[c.off:])
	c.off += n
	return n, nil
}

// Resized receives when the console reports that its screen buffer changed
// size.
func (c *consoleReader) Resized() <-chan struct{} {
	return c.resized
}

// Close ends a Read waiting for input, and has the next fail. Calling it
// again does nothing.
func (c *consoleReader) Close() error {
	if c.closing.Swap(true) {
		return nil
	}
	if err := windows.SetEvent(c.stop); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return windows.CloseHandle(c.stop)
}
