package terminal

import "time"

// FrameStats holds per-frame timing data for profiling. Populated after
// each RootRenderable.Render call when GatherStats is enabled. Access the
// latest values via RootRenderable.LastFrameStats.
type FrameStats struct {
	// Lifecycle is time spent calling OnUpdate on registered widgets.
	Lifecycle time.Duration
	// Update is time spent in the update pass (dirty propagation).
	Update time.Duration
	// Layout is time spent computing the flexbox layout.
	Layout time.Duration
	// Sync is time spent synchronizing layout positions to renderables.
	Sync time.Duration
	// Commands is time spent building the render command list.
	Commands time.Duration
	// Render is time spent executing render commands (all widget RenderSelf calls).
	Render time.Duration
	// BufClear is time spent clearing the cell buffer before rendering.
	BufClear time.Duration
	// Flush is the total flush time (diff + write).
	Flush time.Duration
	// FlushBuild is time spent diffing the buffer and building escape sequences.
	FlushBuild time.Duration
	// FlushWrite is time spent writing escape sequences to stdout.
	FlushWrite time.Duration
	// Total is the wall-clock time for the entire frame.
	Total time.Duration
	// CmdCount is the number of render commands executed this frame.
	CmdCount int
	// NodeCount is the total visible nodes traversed during the update pass.
	NodeCount int
	// FlushBytes is the number of bytes written to the terminal this frame.
	FlushBytes int
}
