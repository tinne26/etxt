package esizer

import "golang.org/x/image/font"
import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt/efixed"

// A [Sizer] with a fixed advance between glyphs.
type FixedSizer struct {
	advance fixed.Int26_6
	buffer  sfnt.Buffer
}

// Sets the fixed advance between characters.
func (self *FixedSizer) SetAdvance(advance int) {
	self.SetAdvanceFract(fixed.Int26_6(advance << 6))
}

// Like [FixedSizer.SetAdvance], but expecting a float64 instead of an int.
func (self *FixedSizer) SetAdvanceFloat(advance float64) {
	self.advance = efixed.FromFloat64RoundToZero(advance)
}

// Like [FixedSizer.SetAdvance], but expcting a fixed.Int26_6 instead of an int.
func (self *FixedSizer) SetAdvanceFract(advance fixed.Int26_6) {
	self.advance = advance
}

// Satisfies the [Sizer] interface.
func (self *FixedSizer) Metrics(font *Font, size fixed.Int26_6) font.Metrics {
	return DefaultMetricsFunc(font, size, &self.buffer)
}

// Satisfies the [Sizer] interface.
func (self *FixedSizer) Advance(*Font, GlyphIndex, fixed.Int26_6) fixed.Int26_6 {
	return self.advance
}

// Satisfies the [Sizer] interface.
func (self *FixedSizer) Kern(*Font, GlyphIndex, GlyphIndex, fixed.Int26_6) fixed.Int26_6 {
	return 0
}
