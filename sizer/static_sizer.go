package sizer

import "strconv"
import . "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt/fract"

var _ Sizer = (*StaticSizer)(nil)

// Must call [StaticSizer.NotifyChange]() to properly update values
// after changing AscentMult, DescentMult or LineGapMult.
type StaticSizer struct {
	AscentMult fract.Unit
	DescentMult fract.Unit
	LineGapMult fract.Unit
	cachedAscent  fract.Unit
	cachedDescent fract.Unit
	cachedLineHeight fract.Unit
}

// Implements [Sizer.Ascent]().
func (self *StaticSizer) Ascent(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedAscent
}

// Implements [Sizer.Descent]().
func (self *StaticSizer) Descent(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedDescent
}

// Implements [Sizer.LineGap]().
func (self *StaticSizer) LineGap(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedLineHeight - self.cachedAscent - self.cachedDescent
}

// Implements [Sizer.LineHeight]().
func (self *StaticSizer) LineHeight(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedLineHeight
}

// Implements [Sizer.LineAdvance]().
func (self *StaticSizer) LineAdvance(*Font, *Buffer, fract.Unit, int) fract.Unit {
	return self.cachedLineHeight
}

// Implements [Sizer.GlyphAdvance]().
func (self *StaticSizer) GlyphAdvance(font *Font, buffer *Buffer, size fract.Unit, g GlyphIndex) fract.Unit {
	advance, err := font.GlyphAdvance(buffer, g, fixed.Int26_6(size), hintingNone)
	if err == nil { return fract.Unit(advance) }
	panic("font.GlyphAdvance(index = " + strconv.Itoa(int(g)) + ") error: " + err.Error())
}

// Implements [Sizer.Kern]().
func (self *StaticSizer) Kern(font *Font, buffer *Buffer, size fract.Unit, g1, g2 GlyphIndex) fract.Unit {
	kern, err := font.Kern(buffer, g1, g2, fixed.Int26_6(size), hintingNone)
	if err == nil { return fract.Unit(kern) }
	if err == ErrNotFound { return 0 }

	msg := "font.Kern failed for glyphs with indices "
	msg += strconv.Itoa(int(g1)) + " and "
	msg += strconv.Itoa(int(g2)) + ": " + err.Error()
	panic(msg)
}

// Implements [Sizer.NotifyChange]().
func (self *StaticSizer) NotifyChange(_ *Font, _ *Buffer, size fract.Unit) {
	self.cachedAscent  = size.MulUp(self.AscentMult)
	self.cachedDescent = size.MulUp(self.DescentMult)
	self.cachedLineHeight = size.MulUp(self.LineGapMult) + self.cachedAscent + self.cachedDescent
}
