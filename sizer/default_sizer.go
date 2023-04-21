package sizer

import "strconv"
import . "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt/fract"

var _ Sizer = (*DefaultSizer)(nil)

// The default [Sizer] used by etxt renderers. For more information
// about sizers, see the documentation of the [Sizer] interface.
type DefaultSizer struct {
	cachedAscent  fract.Unit
	cachedDescent fract.Unit
	cachedLineHeight fract.Unit
	unused fract.Unit
}

// Implements [Sizer.Ascent]().
func (self *DefaultSizer) Ascent(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedAscent
}

// Implements [Sizer.Descent]().
func (self *DefaultSizer) Descent(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedDescent
}

// Implements [Sizer.LineGap]().
func (self *DefaultSizer) LineGap(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedLineHeight - self.cachedAscent - self.cachedDescent
}

// Implements [Sizer.LineHeight]().
func (self *DefaultSizer) LineHeight(*Font, *Buffer, fract.Unit) fract.Unit {
	return self.cachedLineHeight
}

// Implements [Sizer.LineAdvance]().
func (self *DefaultSizer) LineAdvance(*Font, *Buffer, fract.Unit, int) fract.Unit {
	return self.cachedLineHeight
}

// Implements [Sizer.GlyphAdvance]().
func (self *DefaultSizer) GlyphAdvance(font *Font, buffer *Buffer, size fract.Unit, g GlyphIndex) fract.Unit {
	advance, err := font.GlyphAdvance(buffer, g, fixed.Int26_6(size), hintingNone)
	if err == nil { return fract.Unit(advance) }
	panic("font.GlyphAdvance(index = " + strconv.Itoa(int(g)) + ") error: " + err.Error())
}

// Implements [Sizer.Kern]().
func (self *DefaultSizer) Kern(font *Font, buffer *Buffer, size fract.Unit, g1, g2 GlyphIndex) fract.Unit {
	kern, err := font.Kern(buffer, g1, g2, fixed.Int26_6(size), hintingNone)
	if err == nil { return fract.Unit(kern) }
	if err == ErrNotFound { return 0 }

	msg := "font.Kern failed for glyphs with indices "
	msg += strconv.Itoa(int(g1)) + " and "
	msg += strconv.Itoa(int(g2)) + ": " + err.Error()
	panic(msg)
}

// Implements [Sizer.NotifyChange]().
func (self *DefaultSizer) NotifyChange(font *Font, buffer *Buffer, size fract.Unit) {
	metrics, err := font.Metrics(buffer, fixed.Int26_6(size), hintingNone)
	if err != nil { panic("font.Metrics error: " + err.Error()) }
	self.cachedAscent  = fract.Unit(metrics.Ascent)
	self.cachedDescent = fract.Unit(metrics.Descent)
	self.cachedLineHeight = fract.Unit(metrics.Height)
}
