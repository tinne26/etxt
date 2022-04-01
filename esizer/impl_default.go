package esizer

import "golang.org/x/image/font"
import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

// The default sizer used by etxt renderers. For more information
// about sizers, see the documentation of the Sizer interface.
type DefaultSizer struct { buffer sfnt.Buffer }

// Satisfies the sizer interface.
func (self *DefaultSizer) Metrics(font *Font, size fixed.Int26_6) font.Metrics {
	return DefaultMetricsFunc(font, size, &self.buffer)
}

// Satisfies the sizer interface.
func (self *DefaultSizer) Advance(font *Font, glyphIndex GlyphIndex, size fixed.Int26_6) fixed.Int26_6 {
	return DefaultAdvanceFunc(font, glyphIndex, size, &self.buffer)
}

// Satisfies the sizer interface.
func (self *DefaultSizer) Kern(font *Font, prevGlyphIndex GlyphIndex, currGlyphIndex GlyphIndex, size fixed.Int26_6) fixed.Int26_6 {
	return DefaultKernFunc(font, prevGlyphIndex, currGlyphIndex, size, &self.buffer)
}
