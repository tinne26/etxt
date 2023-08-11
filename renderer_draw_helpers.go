package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

func maxInt(a, b int) int {
	if a >= b { return a }
	return b
}

// TODO: LTR and RTL are not really saying enough. If you do ltr processing
//       on ltr iterator, you have ltr. if you do rtl processing on a ltr
//       iterator, you are doing rtl processing. etc. you can always adapt
//       stuff. we need to know what we are doing. like, RtlOnRtlIter
//       maybe just RtlReverse and LtrReverse for reversed algorithms?

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

func (self *Renderer) internalGlyphDraw(target TargetImage, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
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

// expects an quantized position, returns an unquantized position
func (self *Renderer) drawGlyphLTR(target TargetImage, position fract.Point, codePoint rune, iv drawInternalValues) (fract.Point, drawInternalValues) {
	// get glyph index
	currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
					
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

func (self *Renderer) drawGlyphRTL(target TargetImage, position fract.Point, codePoint rune, iv drawInternalValues) (fract.Point, drawInternalValues) {
	// get glyph index
	currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
	
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

func (self *Renderer) drawLineLTR(target TargetImage, position fract.Point, iv drawInternalValues, iterator ltrStringIterator, text string, runeCount int) (fract.Point, drawInternalValues, ltrStringIterator) {
	for i := 0; i < runeCount; i++ {
		codePoint := iterator.Next(text)
		position, iv = self.drawGlyphLTR(target, position, codePoint, iv)
	}
	return position, iv, iterator
}

func (self *Renderer) drawLineRTL(target TargetImage, position fract.Point, iv drawInternalValues, iterator ltrStringIterator, text string, runeCount int) (fract.Point, drawInternalValues, ltrStringIterator) {
	for i := 0; i < runeCount; i++ {
		codePoint := iterator.Next(text)
		position, iv = self.drawGlyphRTL(target, position, codePoint, iv)
	}
	return position, iv, iterator
}
