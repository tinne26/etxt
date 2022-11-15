// The eglyr subpackage defines a [Renderer] struct that's exactly the same
// as the main [etxt.Renderer], but overrides the main methods to operate with
// glyphs instead of strings.
//
// This subpackage is only relevant if you are doing [text shaping] on your own.
// 
// This subpackage is only provided as proof of concept on how to use type
// embedding and the original renderer's TraverseGlyphs() method in order
// to create a more specialized renderer type.
//
// [text shaping]: https://github.com/tinne26/etxt/blob/main/docs/shaping.md
package eglyr

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/emask"
import "golang.org/x/image/math/fixed"

// A type alias to prevent exposing the internal etxt.Renderer embedded
// in the new glyph-specialized renderer.
type internalRenderer = etxt.Renderer

// Like [etxt.Renderer], but with some methods overridden and adapted to
// operate with [etxt.GlyphIndex] instead of strings.
//
// More concretely, the adapted methods are [Renderer.SelectionRect](),
// [Renderer.Draw]() and [Renderer.DrawFract]().
type Renderer struct { internalRenderer }

// Creates a new glyph-specialized renderer with the default vector
// rasterizer. See [NewRenderer]() documentation for more details.
func NewStdRenderer() *Renderer {
	return NewRenderer(&emask.DefaultRasterizer{})
}

// Creates a new glyph-specialized renderer with the given glyph mask
// rasterizer. For the default rasterizer, see [NewStdRenderer]() instead.
//
// After creating a renderer, you must set at least the font and
// the target in order to be able to draw. In most cases, you will
// also want to set a cache handler and a color. Check the setter
// functions for more details on all those.
//
// Renderers are not safe for concurrent use.
func NewRenderer(rasterizer emask.Rasterizer) *Renderer {
	return &Renderer{ *etxt.NewRenderer(rasterizer) }
}

// An alias for [Renderer.SelectionRectGlyphs]().
func (self *Renderer) SelectionRect(glyphIndices []GlyphIndex) etxt.RectSize {
	return self.internalRenderer.SelectionRectGlyphs(glyphIndices)
}

// Draws the given glyphs with the current configuration (font, size, color,
// target, etc). The position at which the glyphs will be drawn depends on
// the given pixel coordinates and the renderer's align (see
// [Renderer.SetAlign]() rules).
//
// The returned value should be ignored except on advanced use-cases
// (refer to [Renderer.Traverse]() documentation).
//
// Missing glyphs in the current font will cause the renderer to panic.
// See [etxt.GetMissingRunes]() if you need to make your system more robust.
//
// Line breaks encoded as \n will be handled automatically.
func (self *Renderer) Draw(glyphIndices []GlyphIndex, x, y int) fixed.Point26_6 {
	fx, fy := fixed.Int26_6(x << 6), fixed.Int26_6(y << 6)
	return self.DrawFract(glyphIndices, fx, fy)
}

// Exactly the same as [Renderer.Draw](), but accepting [fractional pixel] coordinates.
//
// Notice that passing a fractional coordinate won't make the draw operation
// be fractionally aligned by itself, that still depends on the renderer's
// [etxt.QuantizationMode].
//
// [fractional pixel]: https://github.com/tinne26/etxt/blob/main/docs/fixed-26-6.md
func (self *Renderer) DrawFract(glyphIndices []GlyphIndex, x, y fixed.Int26_6) fixed.Point26_6 {
	if len(glyphIndices) == 0 { return fixed.Point26_6{ X: x, Y: y } }

	// traverse glyphs and draw them
	return self.TraverseGlyphs(glyphIndices, fixed.Point26_6{ X: x, Y: y },
		func(currentDot fixed.Point26_6, glyphIndex GlyphIndex) {
			mask := self.LoadGlyphMask(glyphIndex, currentDot)
			self.DefaultDrawFunc(currentDot, mask, glyphIndex)
		})
}
