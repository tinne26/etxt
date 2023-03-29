// The mask subpackage defines the [Rasterizer] interface used within etxt
// and provides multiple ready-to-use implementations.
//
// In this context, "[Rasterizer]" refers to a "glyph mask rasterizer":
// whenever we want to render text on a screen we first have to rasterize
// the individual font glyphs, extracted from font files as outlines
// (sets of lines and curves), and draw them into a raster image (a grid
// of pixels).
//
// In short, this subpackage allows anyone to pick different rasterizers
// or implement their own by targeting the [Rasterizer] interface. This
// opens the door to the creation of cool effects that may modify the
// glyph outlines (e.g.: glyph expansion), the rasterization algorithms
// (e.g.: hinting), the resulting glyph masks (e.g.: blurring) or any
// combination of the previous.
//
// That said, before you jump into the hype train, notice that some of these
// effects can also be achieved (often more easily) at later stages with
// shaders or custom blitting.
package mask
