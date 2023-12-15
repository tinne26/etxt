package sizer

import . "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/fract"

var _ Sizer = (*PaddedAdvanceSizer)(nil)

// Like [PaddedKernSizer], but adds the extra padding in the advance
// instead of the kern.
//
// If you aren't modifying the glyphs, only padding them horizontally,
// use [PaddedKernSizer] instead. This sizer is intended to deal with
// modified glyphs that have actually become wider, like in a faux
// bold process.
type PaddedAdvanceSizer struct {
	defaultSizer
}

// Sets the configurable horizontal padding value.
func (self *PaddedAdvanceSizer) SetPadding(value fract.Unit) {
	self.defaultSizer.unused = value
}

// Returns the configurable horizontal padding value.
func (self *PaddedAdvanceSizer) GetPadding() fract.Unit {
	return self.defaultSizer.unused
}

// Satisfies the [Sizer] interface.
func (self *PaddedAdvanceSizer) GlyphAdvance(font *Font, buffer *Buffer, size fract.Unit, g GlyphIndex) fract.Unit {
	return self.defaultSizer.GlyphAdvance(font, buffer, size, g) + self.defaultSizer.unused
}
