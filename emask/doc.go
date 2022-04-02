// The emask subpackage contains the definition of the Rasterizer
// interface used within etxt, along with multiple ready-to-use
// implementations.
//
// In this context, "Rasterizer" refers to a "glyph mask rasterizer".
// Basically, if we want to render text on a screen we first have to
// rasterize the individual font glyphs, defined as outlines (a set of
// lines and curves), into a raster image (a grid of pixels).
//
// In short, this subpackage allows anyone to pick different rasterizers
// or implement their own by targeting the main Rasterizer interface,
// opening the door to the creation of cool effects that may be achieved
// by modifying the glyph outlines themselves (e.g: glyph expansion),
// modifying the rasterization algorithms (e.g: hinting), modifying the
// resulting glyph masks (e.g: blurring)... or everything at once!
//
// That said, before you jump into the hype train, notice that some of these
// effects can also be achieved (often more easily) at later stages with
// shaders or custom blitting.
package emask
