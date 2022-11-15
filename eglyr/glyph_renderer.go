// The eglyr subpackage defines a [Renderer] struct that behaves like the main
// [etxt.Renderer], but overriding a few methods to operate with glyph indices
// instead of strings.
//
// This subpackage is only relevant if you are doing [text shaping] on your own.
// 
// This subpackage also demonstrates how to use type embedding and the original
// etxt renderer methods in order to create a more specialized renderer (in this
// case, one that improves support for working with glyph indices).
//
// [text shaping]: https://github.com/tinne26/etxt/blob/main/docs/shaping.md
package eglyr

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/emask"
import "golang.org/x/image/math/fixed"

// A type alias to prevent exposing the internal etxt.Renderer embedded
// in the new glyph-specialized renderer.
type internalRenderer = etxt.Renderer

// A renderer just like [etxt.Renderer], but with a few methods overridden
// and adapted to operate with [GlyphIndex] slices instead of strings.
//
// Despite the documentation missing the inherited methods, notice that all
// the property setters and getters available for [etxt.Renderer] are also
// available for this renderer.
type Renderer struct { internalRenderer }

// Creates a new glyph-specialized [Renderer] with the default vector
// rasterizer.
//
// This method is the eglyr equivalent to [etxt.NewStdRenderer]().
func NewStdRenderer() *Renderer {
	return NewRenderer(&emask.DefaultRasterizer{})
}

// Creates a new [Renderer] with the given glyph mask rasterizer.
// For the default rasterizer, see [NewStdRenderer]() instead.
//
// This method is the eglyr equivalent to [etxt.NewRenderer]().
func NewRenderer(rasterizer emask.Rasterizer) *Renderer {
	return &Renderer{ *etxt.NewRenderer(rasterizer) }
}

// An alias for [etxt.Renderer.SelectionRectGlyphs]().
func (self *Renderer) SelectionRect(glyphIndices []GlyphIndex) etxt.RectSize {
	return self.internalRenderer.SelectionRectGlyphs(glyphIndices)
}

// Draws the given glyphs with the current configuration. Glyph indices
// outside the [0, font.NumGlyphs()) range will cause the renderer to panic.
//
// This method is the eglyr equivalent to [etxt.Renderer.Draw]().
func (self *Renderer) Draw(glyphIndices []GlyphIndex, x, y int) fixed.Point26_6 {
	fx, fy := fixed.Int26_6(x << 6), fixed.Int26_6(y << 6)
	return self.DrawFract(glyphIndices, fx, fy)
}

// Same as [Renderer.Draw](), but accepting [fractional pixel] coordinates.
//
// This method is the eglyr equivalent to [etxt.Renderer.DrawFract]().
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
