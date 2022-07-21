package esizer

import "golang.org/x/image/font"
import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt/efixed"

// Like [HorzPaddingSizer], but adds the extra padding in the advance
// instead of the kern.
//
// If you aren't modifying the glyphs, only padding them horizontally,
// use [HorzPaddingSizer] instead. This sizer is intended to deal with
// modified glyphs that have actually become wider.
type AdvancePadSizer struct {
	buffer sfnt.Buffer
	padding fixed.Int26_6
}

// Sets the configurable horizontal padding value, in pixels.
func (self *AdvancePadSizer) SetPadding(value int) {
	self.padding = fixed.Int26_6(value << 6)
}

// Like [AdvancePadSizer.SetPadding], but expecting a fixed.Int26_6 instead
// of an int.
func (self *AdvancePadSizer) SetPaddingFract(value fixed.Int26_6) {
	self.padding = value
}

// Like [AdvancePadSizer.SetPadding], but expecting a float64 instead of
// an int.
func (self *AdvancePadSizer) SetPaddingFloat(value float64) {
	self.padding = efixed.FromFloat64RoundToZero(value)
}

// Satisfies the [Sizer] interface.
func (self *AdvancePadSizer) Metrics(f *Font, size fixed.Int26_6) font.Metrics {
	return DefaultMetricsFunc(f, size, &self.buffer)
}

// Satisfies the [Sizer] interface.
func (self *AdvancePadSizer) Advance(f *Font, glyphIndex GlyphIndex, size fixed.Int26_6) fixed.Int26_6 {
	return DefaultAdvanceFunc(f, glyphIndex, size, &self.buffer) + self.padding
}

// Satisfies the [Sizer] interface.
func (self *AdvancePadSizer) Kern(f *Font, prevGlyphIndex GlyphIndex, currGlyphIndex GlyphIndex, size fixed.Int26_6) fixed.Int26_6 {
	return DefaultKernFunc(f, prevGlyphIndex, currGlyphIndex, size, &self.buffer)
}
