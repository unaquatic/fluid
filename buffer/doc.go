// Package buffer provides [Buffer], the grid of styled [Cell] values that
// widgets draw into and the terminal renderer flushes.
//
// A Buffer draws text, boxes, borders, grids, text surfaces, and pixel data
// as block characters or grayscale ramps. It clips drawing to a stack of
// scissor rectangles, keeps the background beneath a cell written with a
// transparent one, and keeps wide characters and grapheme clusters whole
// across the cells they cover.
package buffer
