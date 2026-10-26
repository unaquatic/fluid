package terminal

import "time"

// FrameStats contains timings and counts from the most recent frame. Fields
// not measured by a rendering path remain zero.
type FrameStats struct {
	// Lifecycle is time spent running the passes added with AddLifecyclePass.
	Lifecycle time.Duration
	// Layout is time spent computing the flexbox layout.
	Layout time.Duration
	// Sync is time spent synchronizing layout positions to renderables.
	Sync time.Duration
	// Commands is time spent building the render command list.
	Commands time.Duration
	// Render is time spent executing widget render commands.
	Render time.Duration
	// BufClear is time spent clearing the cell buffer before rendering.
	BufClear time.Duration
	// Flush is the total time spent building and writing terminal output.
	Flush time.Duration
	// FlushBuild is time spent diffing the buffer and building escape sequences.
	FlushBuild time.Duration
	// FlushWrite is time spent writing escape sequences to stdout.
	FlushWrite time.Duration
	// Total is the wall-clock time for the entire frame.
	Total time.Duration
	// CmdCount is the number of render commands executed.
	CmdCount int
	// NodeCount is the number of visible nodes visited during position sync.
	NodeCount int
	// FlushBytes is the number of terminal-output bytes written.
	FlushBytes int
}
