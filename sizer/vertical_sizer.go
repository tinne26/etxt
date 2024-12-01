package sizer

import "strconv"
import . "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt/fract"

var _ Sizer = (*CustomVertSizer)(nil)

// A sizer that ignores the specific vertical metrics provided by
// the font and instead replaces them with fixed values relative to
// the font size. This can be used to manually control the line
// height for a single font or a small set of fonts.
//
// One must call [CustomVertSizer.NotifyChange]() to properly update
// values after changing AscentMult, DescentMult or LineGapMult.
type CustomVertSizer struct {
	AscentMult       fract.Unit
	DescentMult      fract.Unit
	LineGapMult      fract.Unit
	cachedAscent     fract.Unit
	cachedDescent    fract.Unit
	cachedLineHeight fract.Unit
}

// Satisfies the [Sizer] interface.
func (self *CustomVertSizer) Ascent(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedAscent
}

// Satisfies the [Sizer] interface.
func (self *CustomVertSizer) Descent(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedDescent
}

// Satisfies the [Sizer] interface.
func (self *CustomVertSizer) LineGap(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedLineHeight - self.cachedAscent - self.cachedDescent
}

// Satisfies the [Sizer] interface.
func (self *CustomVertSizer) LineHeight(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedLineHeight
}

// Satisfies the [Sizer] interface.
func (self *CustomVertSizer) LineAdvance(*Font, *Buffer, fract.Unit, int) fract.Unit {
	return self.cachedLineHeight
}

// Satisfies the [Sizer] interface.
func (self *CustomVertSizer) GlyphAdvance(font *Font, buffer *Buffer, size fract.Unit, g GlyphIndex) fract.Unit {
	advance, err := font.GlyphAdvance(buffer, g, fixed.Int26_6(size), hintingNone)
	if err == nil {
		return fract.Unit(advance)
	}
	panic("font.GlyphAdvance(index = " + strconv.Itoa(int(g)) + ") error: " + err.Error())
}

// Satisfies the [Sizer] interface.
func (self *CustomVertSizer) Kern(font *Font, buffer *Buffer, size fract.Unit, g1, g2 GlyphIndex) fract.Unit {
	kern, err := font.Kern(buffer, g1, g2, fixed.Int26_6(size), hintingNone)
	if err == nil {
		return fract.Unit(kern)
	}
	if err == ErrNotFound {
		return 0
	}

	msg := "font.Kern failed for glyphs with indices "
	msg += strconv.Itoa(int(g1)) + " and "
	msg += strconv.Itoa(int(g2)) + ": " + err.Error()
	panic(msg)
}

// Satisfies the [Sizer] interface.
func (self *CustomVertSizer) NotifyChange(_ *Font, _ *Buffer, size fract.Unit) {
	self.cachedAscent = size.MulUp(self.AscentMult)
	self.cachedDescent = size.MulUp(self.DescentMult)
	self.cachedLineHeight = size.MulUp(self.LineGapMult) + self.cachedAscent + self.cachedDescent
}
