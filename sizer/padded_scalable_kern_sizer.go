package sizer

import (
	"github.com/tinne26/etxt/fract"
	. "golang.org/x/image/font/sfnt"
)

var _ Sizer = (*PaddedScalableKernSizer)(nil)

// Similar to [PaddedKernSizer], but instead of taking the padding
// as an absolute value, it uses a value relative to a font size of
// 16px and scales it automatically based on the active font size.
//
// After modifying PaddingAt16px you must call [PaddedScalableKernSizer.NotifyChanges]().
type PaddedScalableKernSizer struct {
	defaultSizer
	PaddingAt16px fract.Unit
}

// Returns the current padding scaled by the given size.
func (self *PaddedScalableKernSizer) GetPaddingAtSize(size fract.Unit) fract.Unit {
	return self.PaddingAt16px.Rescale(16<<6, size)
}

// Satisfies the [Sizer] interface.
func (self *PaddedScalableKernSizer) Kern(font *Font, buffer *Buffer, size fract.Unit, g1, g2 GlyphIndex) fract.Unit {
	return self.defaultSizer.Kern(font, buffer, size, g1, g2) + self.defaultSizer.unused
}

// Satisfies the [Sizer] interface.
func (self *PaddedScalableKernSizer) NotifyChange(font *Font, buffer *Buffer, size fract.Unit) {
	self.defaultSizer.unused = self.GetPaddingAtSize(size)
	self.defaultSizer.NotifyChange(font, buffer, size)
}
