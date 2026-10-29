// Package render defines the scene graph that fluid draws.
//
// Every widget is a [Renderable]. Most embed [BaseRenderable], which holds a
// widget's children, layout node, visibility, focus, input handlers, and the
// damage it needs repainted. A renderable reaches its renderer through a
// [Host], for example to request a frame or place the cursor, and
// [Selection] tracks text selected across renderables.
//
// # The tree
//
// [BaseRenderable.Add], [BaseRenderable.Remove], and
// [BaseRenderable.InsertBefore] change a node's children.
// [BaseRenderable.Renderable] and [BaseRenderable.FindDescendantByID] find
// nodes by the IDs that [WithID] and widget options such as BoxWithID set, and
// [BaseRenderable.SetVisible] hides a node but keeps its space. Handler
// fields, such as [BaseRenderable.OnMouseDownHandler] and
// [BaseRenderable.OnKeyDownHandler], respond to input without a new type.
//
// For each frame, the renderer lays the tree out and then draws it depth
// first: each visible node's RenderSelf draws into a [buffer.Buffer] before
// its children do, so children paint over their parent and later siblings
// over earlier ones. Only the region that changed since the last frame is
// drawn again.
//
// # Writing a widget
//
// A widget embeds *BaseRenderable and overrides the methods it needs: usually
// RenderSelf to draw and, if it is focusable, HandleKeyPress to take keys. A
// widget that sizes itself sets a [layout.MeasureFunc] on its layout node.
//
// RenderSelf draws the widget alone, at the position and size that X, Y,
// Width, and Height report; the renderer draws its children afterwards. A
// change that can alter the widget's size marks its layout node dirty, so
// that layout measures it again, and every change calls RequestRender, so
// that the next frame draws the widget again. The package example writes a
// widget that does each of these.
package render
