package terminal

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// WidthProbe selects how the renderer establishes terminal cell widths.
type WidthProbe uint8

const (
	// WidthProbeOff keeps the built-in Unicode width tables. This is the
	// default: a library should not hold a conversation with the terminal
	// unless it is asked to.
	WidthProbeOff WidthProbe = iota

	// WidthProbeAuto measures Config.WidthProbeSamples at startup and reports
	// the result via Renderer.ProbedWidths. Measuring cannot change behaviour
	// on a terminal that already agrees with the built-in tables.
	WidthProbeAuto
)

// String implements fmt.Stringer.
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
	// probeTimeout bounds the wait for cursor position reports. A terminal
	// that does not implement DSR must cost a moment at startup, never a hang.
	probeTimeout = 250 * time.Millisecond

	// probeDrainQuiet bounds how long a failed probe waits for late replies
	// before concluding the terminal has stopped talking.
	probeDrainQuiet = 50 * time.Millisecond
)

// ErrProbeUnsupported reports that terminal measurement was not possible, so
// the built-in width tables should be kept.
var ErrProbeUnsupported = errors.New("terminal: width probe unsupported")

// ErrProbeTimeout reports that the terminal did not answer the cursor position
// request in time, which is how a terminal without DSR support presents.
var ErrProbeTimeout = errors.New("terminal: width probe timed out")

// ProbeWidths measures how many columns the terminal advances for each sample.
//
// Every sample is written at the start of a cleared line followed by a Device
// Status Report (CSI 6n); the terminal replies with a cursor position report
// giving the column it ended on. All samples go out in a single write because
// terminals process input sequentially and queue their replies, making this
// one round-trip rather than one per sample.
//
// Call this only while nothing else reads stdin — during startup, before the
// input loop begins. Bytes that are not cursor position reports are discarded,
// so a keystroke landing mid-probe is swallowed rather than misparsed.
//
// A non-nil error means the terminal did not answer and the caller must keep
// the built-in width tables.
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
		// Replies still in flight when we gave up would be read by the input
		// loop instead. A cursor position report is CSI row ; col R, and the
		// key parser resolves CSI 1 ; n R to a modified F3 — so abandoning
		// them mid-probe injects keystrokes the user never typed.
		drainInput(f, probeDrainQuiet)
		return nil, err
	}
	return widths, nil
}

// drainInput discards whatever the terminal is still sending, stopping once it
// has been quiet for quiet or the overall budget is spent.
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

// writeProbes emits every sample followed by a cursor position request, each
// on a freshly cleared line so the samples never accumulate on screen.
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

// readProbeReplies collects want cursor position reports and converts each
// one-based column into the width the sample occupied. Reads only once poll
// reports data, so a silent terminal ends at the deadline instead of blocking.
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

// parseCPR extracts the columns from cursor position reports (CSI row;col R)
// in buf. Anything that is not a report is discarded; rest holds a trailing
// partial report awaiting more bytes.
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
			return cols, bytes.Clone(buf[i:]) // report split across reads
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

// cprColumn pulls the column out of a "row;col" parameter list.
func cprColumn(params []byte) (int, bool) {
	k := bytes.IndexByte(params, ';')
	if k < 0 {
		return 0, false
	}
	c, err := strconv.Atoi(string(params[k+1:]))
	if err != nil || c < 1 {
		return 0, false
	}
	return c, true
}
