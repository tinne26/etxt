package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// Precondition: lineBreakX is properly quantized already. lineBreakNth has been
// preincremented, for example with drawInternalValues.increaseLineBreakNth()
func (self *Renderer) advanceLine(position fract.Point, lineBreakX fract.Unit, lineBreakNth int) fract.Point {
	prevFractY := position.Y.FractShift()
	position.X  = lineBreakX
	position.Y += self.getOpLineAdvance(lineBreakNth)
	position.Y  = position.Y.QuantizeUp(fract.Unit(self.state.vertQuantization))
	if self.cacheHandler != nil && position.Y.FractShift() != prevFractY {
		self.cacheHandler.NotifyFractChange(position)
	}

	return position
}

func (self *Renderer) internalGlyphDraw(target Target, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
	if self.customDrawFn != nil {
		self.customDrawFn(target, glyphIndex, origin)
	} else {
		mask := self.loadGlyphMask(self.state.activeFont, glyphIndex, origin)
		self.defaultDrawFunc(target, origin, mask)
	}
}

type drawInternalValues struct {
	prevFractX fract.Unit
	prevGlyphIndex sfnt.GlyphIndex
	lineBreakNth int
}

func (self *drawInternalValues) increaseLineBreakNth() {
	self.lineBreakNth = maxInt(1, self.lineBreakNth + 1)
}

func (self *Renderer) drawRuneLTR(target Target, position fract.Point, codePoint rune, iv drawInternalValues) (fract.Point, drawInternalValues) {
	glyph := self.getGlyphIndex(self.state.activeFont, codePoint)
	return self.drawGlyphLTR(target, position, glyph, iv)
}

// expects a quantized position, returns an unquantized position
func (self *Renderer) drawGlyphLTR(target Target, position fract.Point, currGlyphIndex sfnt.GlyphIndex, iv drawInternalValues) (fract.Point, drawInternalValues) {
	// apply kerning unless coming from line break
	if iv.lineBreakNth != 0 {
		iv.lineBreakNth = 0
	} else {
		position.X += self.getOpKernBetween(iv.prevGlyphIndex, currGlyphIndex)
	}
	position.X = position.X.QuantizeUp(fract.Unit(self.state.horzQuantization))

	if position.X.FractShift() != iv.prevFractX {
		iv.prevFractX = position.X.FractShift()
		self.cacheHandler.NotifyFractChange(position)
	}

	// draw glyph
	self.internalGlyphDraw(target, currGlyphIndex, position)

	// advance
	position.X += self.getOpAdvance(currGlyphIndex)

	iv.prevGlyphIndex = currGlyphIndex
	return position, iv
}

func (self *Renderer) drawRuneRTL(target Target, position fract.Point, codePoint rune, iv drawInternalValues) (fract.Point, drawInternalValues) {
	glyph := self.getGlyphIndex(self.state.activeFont, codePoint)
	return self.drawGlyphRTL(target, position, glyph, iv)
}

// expects a quantized position, returns a quantized position
func (self *Renderer) drawGlyphRTL(target Target, position fract.Point, currGlyphIndex sfnt.GlyphIndex, iv drawInternalValues) (fract.Point, drawInternalValues) {
	// advance
	position.X -= self.getOpAdvance(currGlyphIndex)

	// apply kerning unless coming from line break
	if iv.lineBreakNth != 0 {
		iv.lineBreakNth = 0
	} else {
		position.X -= self.getOpKernBetween(currGlyphIndex, iv.prevGlyphIndex)
	}
	position.X = position.X.QuantizeUp(fract.Unit(self.state.horzQuantization))

	if position.X.FractShift() != iv.prevFractX {
		iv.prevFractX = position.X.FractShift()
		self.cacheHandler.NotifyFractChange(position)
	}
	
	// draw glyph
	self.internalGlyphDraw(target, currGlyphIndex, position)

	iv.prevGlyphIndex = currGlyphIndex
	return position, iv
}

func (self *Renderer) helperDrawLineLTR(target Target, position fract.Point, iv drawInternalValues, iterator ltrStringIterator, text string, runeCount int) (fract.Point, drawInternalValues, ltrStringIterator) {
	for i := 0; i < runeCount; i++ {
		codePoint := iterator.Next(text)
		position, iv = self.drawRuneLTR(target, position, codePoint, iv)
	}
	return position, iv, iterator
}

func (self *Renderer) helperDrawLineReverseLTR(target Target, position fract.Point, iv drawInternalValues, iterator ltrStringIterator, text string, runeCount int) (fract.Point, drawInternalValues, ltrStringIterator) {
	for i := 0; i < runeCount; i++ {
		codePoint := iterator.Next(text)
		position, iv = self.drawRuneRTL(target, position, codePoint, iv)
	}
	return position, iv, iterator
}

func (self *Renderer) helperDrawLineReverseRTL(target Target, position fract.Point, iv drawInternalValues, iterator rtlStringIterator, text string, runeCount int) (fract.Point, drawInternalValues, rtlStringIterator) {
	for i := 0; i < runeCount; i++ {
		codePoint := iterator.Next(text)
		position, iv = self.drawRuneLTR(target, position, codePoint, iv)
	}
	return position, iv, iterator
}

func (self *Renderer) helperDrawLineRTL(target Target, position fract.Point, iv drawInternalValues, iterator rtlStringIterator, text string, runeCount int) (fract.Point, drawInternalValues, rtlStringIterator) {
	for i := 0; i < runeCount; i++ {
		codePoint := iterator.Next(text)
		position, iv = self.drawRuneRTL(target, position, codePoint, iv)
	}
	return position, iv, iterator
}
