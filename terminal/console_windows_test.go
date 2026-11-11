//go:build windows

package terminal

import (
	"errors"
	"os"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procAllocConsole      = windows.NewLazySystemDLL("kernel32.dll").NewProc("AllocConsole")
	procWriteConsoleInput = windows.NewLazySystemDLL("kernel32.dll").NewProc("WriteConsoleInputW")
)

// testConsole returns an input handle of the test's console, which it gets
// one of when it has none, emptied and in a mode that keeps the records
// written to it as they are, or skips the test when it gets no console.
func testConsole(t *testing.T) windows.Handle {
	t.Helper()
	name, err := windows.UTF16PtrFromString("CONIN$")
	if err != nil {
		t.Fatal(err)
	}
	open := func() (windows.Handle, error) {
		return windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	}
	h, err := open()
	if err != nil {
		procAllocConsole.Call()
		if h, err = open(); err != nil {
			t.Skipf("no console: %v", err)
		}
	}
	t.Cleanup(func() { windows.CloseHandle(h) })
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		t.Skipf("no console: %v", err)
	}
	if err := windows.SetConsoleMode(h, windows.ENABLE_WINDOW_INPUT); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { windows.SetConsoleMode(h, mode) })
	if err := windows.FlushConsoleInputBuffer(h); err != nil {
		t.Fatal(err)
	}
	return h
}

// TestConsoleReaderReadsRecords writes records to a console's input, and
// checks that a consoleReader reads them as the bytes they stand for, Ctrl-Z
// among them, and reports the resize among them.
func TestConsoleReaderReadsRecords(t *testing.T) {
	h := testConsole(t)
	records := append(typed("a\x1ab"), keyRecord('b', false), ConsoleRecord{EventType: ConsoleBufferSizeEvent})
	var written uint32
	r, _, err := procWriteConsoleInput.Call(uintptr(h), uintptr(unsafe.Pointer(&records[0])), uintptr(len(records)), uintptr(unsafe.Pointer(&written)))
	if r == 0 || int(written) != len(records) {
		t.Fatalf("WriteConsoleInput wrote %d of %d records: %v", written, len(records), err)
	}

	c, err := newConsoleReader(h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	var got []byte
	buf := make([]byte, 8)
	for len(got) < 3 {
		n, err := c.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, buf[:n]...)
	}
	if string(got) != "a\x1ab" {
		t.Errorf("read %q; want %q", got, "a\x1ab")
	}
	select {
	case <-c.Resized():
	default:
		t.Error("the resize was not reported")
	}
}

// TestConsoleReaderClose checks that Close ends a Read waiting for input,
// that a Read after it fails at once, and that closing again does nothing.
// An event that is never signaled stands in for the console input.
func TestConsoleReaderClose(t *testing.T) {
	h, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { windows.CloseHandle(h) })
	c, err := newConsoleReader(h)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := c.Read(make([]byte, 8))
		done <- err
	}()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, os.ErrClosed) {
			t.Errorf("waiting Read = %v; want os.ErrClosed", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not end the waiting Read")
	}
	if _, err := c.Read(make([]byte, 8)); !errors.Is(err, os.ErrClosed) {
		t.Errorf("Read after Close = %v; want os.ErrClosed", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("second Close = %v; want nil", err)
	}
}
