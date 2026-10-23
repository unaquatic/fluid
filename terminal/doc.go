// Package terminal runs fluid in a terminal.
//
// A [Renderer] owns the scene's root and its event loop, which applies tree
// changes, lays out and draws each frame, and flushes only the cells that
// changed: inline, into the terminal's normal screen and scrollback, or
// fullscreen, on the alternate screen. [Backend] writes the terminal's escape
// sequences, and [DetectColorMode] picks 24-bit, 256, or 16 colors from the
// environment, which FLUID_COLOR overrides.
package terminal
