package esizer

import "golang.org/x/image/font"
import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

// A default [Sizer] without kerning (the kern function always returns 0).
type NoKernSizer struct { buffer sfnt.Buffer }

// Satisfies the [Sizer] interface.
func (self *NoKernSizer) Metrics(font *Font, size fixed.Int26_6) font.Metrics {
	return DefaultMetricsFunc(font, size, &self.buffer)
}

// Satisfies the [Sizer] interface.
func (self *NoKernSizer) Advance(font *Font, glyphIndex GlyphIndex, size fixed.Int26_6) fixed.Int26_6 {
	return DefaultAdvanceFunc(font, glyphIndex, size, &self.buffer)
}

// Satisfies the [Sizer] interface.
func (self *NoKernSizer) Kern(*Font, GlyphIndex, GlyphIndex, fixed.Int26_6) fixed.Int26_6 {
	return 0
}
