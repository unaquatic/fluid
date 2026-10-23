// Package input turns terminal input into events.
//
// A [Parser] reads raw terminal bytes into [KeyEvent], [MouseEvent], and
// [PasteEvent] values, an [EventBus] lets handlers observe events and prevent
// their default actions, and [KeyBinding] tables map key events to named
// actions with [ResolveAction].
package input
