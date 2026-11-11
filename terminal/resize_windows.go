//go:build windows

package terminal

// resizeNotice is what tells that the console may have changed size: a
// resize that it reported among its input records.
type resizeNotice = struct{}

// notifyResize returns the channel on which the console input that Setup
// opened reports resizes, and a func that does nothing, as the channel
// needs no stopping. Without a console input, the channel is nil and never
// receives.
func (t *Backend) notifyResize() (<-chan resizeNotice, func()) {
	if t.console == nil {
		return nil, func() {}
	}
	return t.console.Resized(), func() {}
}
