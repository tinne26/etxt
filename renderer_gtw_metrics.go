package etxt

import (
	"github.com/tinne26/etxt/fract"
	"golang.org/x/image/font/sfnt"
)

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to metric functions for a [Renderer].
//
// In general, this type is used through method chaining:
//
//	renderer.Metrics().Advance()
//
// This gateway simplifies access to common metrics, which otherwise
// can be tedious to request from the sizer itself:
//
//	font := renderer.GetFont()
//	buffer := renderer.GetBuffer()
//	size := renderer.Fract().GetScaledSize()
//	ascent := renderer.Sizer().Ascent(font, buffer, size)
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9#Renderer
type RendererMetrics Renderer

// [Gateway] to [RendererMetrics] functionality.
//
// [Gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9#Renderer
func (self *Renderer) Metrics() *RendererMetrics {
	return (*RendererMetrics)(self)
}

// Ascent returns the font ascent for the current renderer configuration.
// This is shorthand for [Renderer.Sizer]().Ascent(...).
func (self *RendererMetrics) Ascent() fract.Unit {
	return (*Renderer)(self).getOpAscent()
}

// CapHeight returns the reference height of a capital character for the
// current font and size.
func (self *RendererMetrics) CapHeight() fract.Unit {
	return (*Renderer)(self).getOpCapHeight()
}

// MidHeight returns the reference height of a lowercase character for the
// current font and size.
func (self *RendererMetrics) MidHeight() fract.Unit {
	return (*Renderer)(self).getOpCapHeight()
}

// Descent returns the font descent for the current renderer configuration.
// This is shorthand for [Renderer.Sizer]().Descent(...).
func (self *RendererMetrics) Descent() fract.Unit {
	return (*Renderer)(self).getOpDescent()
}

// LineHeight returns ascent + descent + lineGap for the current renderer
// configuration. This is shorthand for [Renderer.Sizer]().LineHeight(...).
func (self *RendererMetrics) LineHeight() fract.Unit {
	return (*Renderer)(self).getOpLineHeight()
}

// Advance returns the requested glyph's advance under the current renderer
// configuration. This is shorthand for [Renderer.Sizer]().Advance(...).
func (self *RendererMetrics) Advance(glyphIndex sfnt.GlyphIndex) fract.Unit {
	return (*Renderer)(self).getOpAdvance(glyphIndex)
}
