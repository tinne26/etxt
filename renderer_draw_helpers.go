package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// Precondition: lineBreakX is properly quantized already.
func (self *Renderer) advanceLine(position fract.Point, lineBreakX fract.Unit, lineBreakNth int) (fract.Point, int) {
	prevFractX, prevFractY := position.X.FractShift(), position.Y.FractShift()
	position.X  = lineBreakX
	position.Y += self.getOpLineAdvance(lineBreakNth)
	position.Y  = position.Y.QuantizeUp(fract.Unit(self.state.vertQuantization))
	if position.Y.FractShift() != prevFractY || position.X.FractShift() != prevFractX {
		if self.cacheHandler != nil {
			self.cacheHandler.NotifyFractChange(position)
		}
	}

	return position, maxInt(1, lineBreakNth + 1)
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

func (self *Renderer) drawGlyphLTR(target TargetImage, position fract.Point, codePoint rune, iv drawInternalValues) (fract.Point, drawInternalValues) {
	// get glyph index
	currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
					
	// apply kerning unless coming from line break
	if iv.lineBreakNth != 0 {
		iv.lineBreakNth = 0
	} else {
		position.X += self.getOpKernBetween(iv.prevGlyphIndex, currGlyphIndex)
		position.X = position.X.QuantizeUp(fract.Unit(self.state.horzQuantization))
	}

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

	if position.X.FractShift() != iv.prevFractX {
		iv.prevFractX = position.X.FractShift()
		self.cacheHandler.NotifyFractChange(position)
	}
	
	// draw glyph
	self.internalGlyphDraw(target, currGlyphIndex, position)

	// apply kerning unless coming from line break
	if iv.lineBreakNth != 0 {
		iv.lineBreakNth = 0
	} else {
		position.X -= self.getOpKernBetween(iv.prevGlyphIndex, currGlyphIndex)
		position.X = position.X.QuantizeUp(fract.Unit(self.state.horzQuantization))
	}	

	iv.prevGlyphIndex = currGlyphIndex
	return position, iv
}
