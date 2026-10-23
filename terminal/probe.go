package terminal

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// WidthProbe controls whether startup measures terminal cell widths.
type WidthProbe uint8

const (
	// WidthProbeOff uses the built-in Unicode width tables without querying the
	// terminal.
	WidthProbeOff WidthProbe = iota

	// WidthProbeAuto measures Config.WidthProbeSamples during setup and stores
	// the results for Renderer.ProbedWidths.
	WidthProbeAuto
)

// String returns p's name.
func (p WidthProbe) String() string {
	switch p {
	case WidthProbeOff:
		return "off"
	case WidthProbeAuto:
		return "auto"
	}
	return fmt.Sprintf("WidthProbe(%d)", uint8(p))
}

// Valid reports whether p is a defined WidthProbe.
func (p WidthProbe) Valid() bool { return p <= WidthProbeAuto }

const (
	// probeTimeout bounds the complete cursor-report exchange.
	probeTimeout = 250 * time.Millisecond

	// probeDrainQuiet is the quiet interval that ends cleanup after a failed probe.
	probeDrainQuiet = 50 * time.Millisecond
)

// ErrProbeUnsupported indicates that terminal width measurement is unavailable.
var ErrProbeUnsupported = errors.New("terminal: width probe unsupported")

// ErrProbeTimeout indicates that the terminal did not answer before the probe
// deadline.
var ErrProbeTimeout = errors.New("terminal: width probe timed out")

// ProbeWidths returns the number of terminal columns occupied by each sample.
//
// ProbeWidths writes each sample at column zero followed by a Device Status
// Report request. It sends all requests in one write and derives widths from
// the returned cursor columns.
//
// Call ProbeWidths before the input loop starts. It discards bytes that are not
// cursor position reports, including input received during the probe.
//
// A non-nil error leaves callers responsible for retaining built-in widths.
func (t *Backend) ProbeWidths(samples []string) ([]int, error) {
	if len(samples) == 0 {
		return nil, nil
	}
	f, ok := t.stdin.(*os.File)
	if !ok {
		return nil, ErrProbeUnsupported
	}

	if err := t.writeProbes(samples); err != nil {
		return nil, err
	}

	widths, err := readProbeReplies(f, len(samples), time.Now().Add(probeTimeout))
	if err != nil {
		// Drain late cursor reports before the key parser starts consuming input.
		drainInput(f, probeDrainQuiet)
		return nil, err
	}
	return widths, nil
}

// drainInput discards pending terminal input until the quiet interval or total
// probe deadline expires.
func drainInput(f *os.File, quiet time.Duration) {
	fd := int(f.Fd())
	overall := time.Now().Add(probeTimeout)
	buf := make([]byte, 256)

	for time.Now().Before(overall) {
		ready, err := waitReadable(fd, time.Now().Add(quiet))
		if err != nil || !ready {
			return
		}
		if _, err := f.Read(buf); err != nil {
			return
		}
	}
}

// writeProbes writes each sample from column zero on a cleared line, requests
// its cursor position, and clears the line afterward.
func (t *Backend) writeProbes(samples []string) error {
	var out bytes.Buffer
	for _, s := range samples {
		out.WriteString("\r\x1b[2K")
		out.WriteString(s)
		out.WriteString("\x1b[6n")
	}
	out.WriteString("\r\x1b[2K")
	_, err := t.WriteOut(out.Bytes())
	return err
}

// readProbeReplies collects want cursor position reports before deadline and
// converts their one-based columns to sample widths.
func readProbeReplies(f *os.File, want int, deadline time.Time) ([]int, error) {
	fd := int(f.Fd())
	widths := make([]int, 0, want)
	var pending []byte
	buf := make([]byte, 256)

	for len(widths) < want {
		ready, err := waitReadable(fd, deadline)
		if err != nil {
			return nil, fmt.Errorf("terminal: width probe: %w", err)
		}
		if !ready {
			return nil, ErrProbeTimeout
		}
		n, err := f.Read(buf)
		if n > 0 {
			var cols []int
			cols, pending = parseCPR(append(pending, buf[:n]...))
			for _, c := range cols {
				widths = append(widths, c-1)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("terminal: width probe: %w", err)
		}
	}
	return widths, nil
}

// parseCPR extracts columns from CSI row;column R reports. It discards unrelated
// input and returns an incomplete trailing report in rest.
func parseCPR(buf []byte) (cols []int, rest []byte) {
	for {
		i := bytes.Index(buf, []byte("\x1b["))
		if i < 0 {
			if bytes.HasSuffix(buf, []byte("\x1b")) {
				return cols, bytes.Clone(buf[len(buf)-1:])
			}
			return cols, nil
		}
		j := i + 2
		for j < len(buf) && (buf[j] == ';' || (buf[j] >= '0' && buf[j] <= '9')) {
			j++
		}
		if j >= len(buf) {
			return cols, bytes.Clone(buf[i:]) // Preserve a report split across reads.
		}
		if buf[j] != 'R' {
			buf = buf[j+1:]
			continue
		}
		if c, ok := cprColumn(buf[i+2 : j]); ok {
			cols = append(cols, c)
		}
		buf = buf[j+1:]
	}
}

// cprColumn returns the positive column from a row;column parameter list.
func cprColumn(params []byte) (int, bool) {
	_, after, ok := bytes.Cut(params, []byte{';'})
	if !ok {
		return 0, false
	}
	c, err := strconv.Atoi(string(after))
	if err != nil || c < 1 {
		return 0, false
	}
	return c, true
}
