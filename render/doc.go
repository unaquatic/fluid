// Package render defines the scene graph that fluid draws.
//
// Every widget is a [Renderable]. Most embed [BaseRenderable], which holds a
// widget's children, layout node, visibility, focus, input handlers, and the
// damage it needs repainted. A renderable reaches its renderer through a
// [Host], for example to request a frame or place the cursor, and
// [Selection] tracks text selected across renderables.
package render
