package sizer

import . "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/fract"

var _ Sizer = (*PaddedKernSizer)(nil)

// A [Sizer] that behaves like the default one, but with a configurable
// horizontal padding factor that's added to the kern between glyphs.
//
// See also [PaddedScalableKernSizer] if you need to deal with scalable
// text.
type PaddedKernSizer struct {
	defaultSizer
}

// Sets the configurable horizontal kern padding value.
func (self *PaddedKernSizer) SetPadding(value fract.Unit) {
	self.defaultSizer.unused = value
}

// Returns the configurable horizontal kern padding value.
func (self *PaddedKernSizer) GetPadding() fract.Unit {
	return self.defaultSizer.unused
}

// Satisfies the [Sizer] interface.
func (self *PaddedKernSizer) Kern(font *Font, buffer *Buffer, size fract.Unit, g1, g2 GlyphIndex) fract.Unit {
	return self.defaultSizer.Kern(font, buffer, size, g1, g2) + self.defaultSizer.unused
}
