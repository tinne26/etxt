package esizer

import "golang.org/x/image/font"
import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt/efixed"

// A [Sizer] that behaves like the default one, but with a configurable
// horizontal padding factor that's added to the kern between glyphs.
type HorzPaddingSizer struct {
	horzPadding fixed.Int26_6
	buffer sfnt.Buffer
}

// Sets the configurable horizontal padding value, in pixels.
func (self *HorzPaddingSizer) SetHorzPadding(value int) {
	self.horzPadding = fixed.Int26_6(value << 6)
}

// Like [HorzPaddingSizer.SetHorzPadding], but expecting a fixed.Int26_6
// instead of an int.
func (self *HorzPaddingSizer) SetHorzPaddingFract(value fixed.Int26_6) {
	self.horzPadding = value
}

// Like [HorzPaddingSizer.SetHorzPadding], but expecting a float64 instead
// of an int.
func (self *HorzPaddingSizer) SetHorzPaddingFloat(value float64) {
	self.horzPadding = efixed.FromFloat64RoundToZero(value)
}

// Satisfies the [Sizer] interface.
func (self *HorzPaddingSizer) Metrics(f *Font, size fixed.Int26_6) font.Metrics {
	return DefaultMetricsFunc(f, size, &self.buffer)
}

// Satisfies the [Sizer] interface.
func (self *HorzPaddingSizer) Advance(f *Font, glyphIndex GlyphIndex, size fixed.Int26_6) fixed.Int26_6 {
	return DefaultAdvanceFunc(f, glyphIndex, size, &self.buffer)
}

// Satisfies the [Sizer] interface.
func (self *HorzPaddingSizer) Kern(f *Font, prevGlyphIndex GlyphIndex, currGlyphIndex GlyphIndex, size fixed.Int26_6) fixed.Int26_6 {
	return DefaultKernFunc(f, prevGlyphIndex, currGlyphIndex, size, &self.buffer) + self.horzPadding
}
