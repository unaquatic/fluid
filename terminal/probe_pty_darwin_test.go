//go:build darwin

package terminal

import (
	"os"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func ioctlPtr(fd int, req uint, arg unsafe.Pointer) error {
	_, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(arg))
	if e != 0 {
		return e
	}
	return nil
}

// These tests use Darwin-specific pseudo-terminal ioctls.

// openPTY returns a connected pseudo-terminal master and slave.
func openPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no /dev/ptmx: %v", err)
	}
	if err := unix.IoctlSetInt(int(m.Fd()), unix.TIOCPTYGRANT, 0); err != nil {
		t.Skipf("grantpt: %v", err)
	}
	if err := unix.IoctlSetInt(int(m.Fd()), unix.TIOCPTYUNLK, 0); err != nil {
		t.Skipf("unlockpt: %v", err)
	}
	var nameBuf [128]byte
	if err := ioctlPtr(int(m.Fd()), unix.TIOCPTYGNAME, unsafe.Pointer(&nameBuf[0])); err != nil {
		t.Skipf("ptsname: %v", err)
	}
	name := string(nameBuf[:])
	if i := strings.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	s, err := os.OpenFile(name, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("open slave: %v", err)
	}
	// Cursor reports contain no newline, so the slave must use raw mode.
	old, err := term.MakeRaw(int(s.Fd()))
	if err != nil {
		t.Skipf("makeraw: %v", err)
	}
	t.Cleanup(func() { term.Restore(int(s.Fd()), old); s.Close(); m.Close() })
	return m, s
}

// TestProbeWidths_AgainstPTY verifies width probing through a blocking terminal
// file descriptor.
func TestProbeWidths_AgainstPTY(t *testing.T) {
	master, slave := openPTY(t)

	b := NewBackend(slave, slave)

	// Emulate terminal replies with chosen cursor columns.
	widthsWanted := []int{1, 2, 3}
	go func() {
		buf := make([]byte, 4096)
		seen := 0
		for seen < len(widthsWanted) {
			n, err := master.Read(buf)
			if err != nil {
				return
			}
			for _, c := range buf[:n] {
				if c != 'n' {
					continue // only CSI 6n ends in 'n'
				}
				if seen < len(widthsWanted) {
					master.WriteString("\x1b[1;" + itoa(widthsWanted[seen]+1) + "R")
					seen++
				}
			}
		}
	}()

	got, err := b.ProbeWidths([]string{"A", "世", "abc"})
	if err != nil {
		t.Fatalf("ProbeWidths: %v", err)
	}
	for i, w := range widthsWanted {
		if got[i] != w {
			t.Errorf("width %d = %d, want %d", i, got[i], w)
		}
	}
}

// TestProbeWidths_SilentTerminal verifies that an unanswered DSR request times
// out.
func TestProbeWidths_SilentTerminal(t *testing.T) {
	_, slave := openPTY(t)
	b := NewBackend(slave, slave)

	start := time.Now()
	_, err := b.ProbeWidths([]string{"A"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout from a silent terminal")
	}
	if elapsed > 2*time.Second {
		t.Errorf("probe took %v; must be bounded by the deadline", elapsed)
	}
	t.Logf("silent terminal gave up after %v with %v", elapsed, err)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// residue returns the number of bytes waiting on a blocking pseudo-terminal.
func residue(t *testing.T, f *os.File, within time.Duration) []byte {
	t.Helper()
	ready, err := waitReadable(int(f.Fd()), time.Now().Add(within))
	if err != nil || !ready {
		return nil
	}
	var buf [256]byte
	n, _ := f.Read(buf[:])
	return buf[:n]
}

// TestProbeWidths_DrainsLateReplies verifies that timed-out cursor reports do
// not remain for the input loop.
func TestProbeWidths_DrainsLateReplies(t *testing.T) {
	master, slave := openPTY(t)
	b := NewBackend(slave, slave)

	// Reply only after the probe deadline.
	go func() {
		buf := make([]byte, 4096)
		if _, err := master.Read(buf); err != nil {
			return
		}
		time.Sleep(probeTimeout + 20*time.Millisecond)
		master.WriteString("\x1b[1;3R\x1b[1;3R")
	}()

	if _, err := b.ProbeWidths([]string{"A", "B"}); err == nil {
		t.Fatal("expected the probe to time out")
	}
	if left := residue(t, slave, 200*time.Millisecond); len(left) > 0 {
		t.Errorf("%d bytes left for the input loop to misread: %q", len(left), left)
	}
}

// TestProbeWidths_SuccessLeavesNoResidue verifies that successful probing
// consumes every reply.
func TestProbeWidths_SuccessLeavesNoResidue(t *testing.T) {
	master, slave := openPTY(t)
	b := NewBackend(slave, slave)

	go func() {
		buf := make([]byte, 4096)
		seen := 0
		for seen < 2 {
			n, err := master.Read(buf)
			if err != nil {
				return
			}
			for _, c := range buf[:n] {
				if c == 'n' && seen < 2 {
					master.WriteString("\x1b[1;2R")
					seen++
				}
			}
		}
	}()

	if _, err := b.ProbeWidths([]string{"A", "B"}); err != nil {
		t.Fatalf("ProbeWidths: %v", err)
	}
	if left := residue(t, slave, 100*time.Millisecond); len(left) > 0 {
		t.Errorf("%d bytes left after a successful probe: %q", len(left), left)
	}
}
