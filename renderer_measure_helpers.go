package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// Preconditions: font and sizer are not nil
func (self *Renderer) helperMeasureHeight(text string) fract.Unit {
	if text == "" { return 0 }

	// set up traversal variables
	var height fract.Unit
	var lineBreakNth int
	var lineBreaksOnly bool = true
	vertQuant := fract.Unit(self.state.vertQuantization)

	for _, codePoint := range text {
		if codePoint == '\n' {
			lineBreakNth += 1
			height = (height + self.getOpLineAdvance(lineBreakNth)).QuantizeUp(vertQuant)
		} else {
			lineBreaksOnly = false
			lineBreakNth = 0
		}
	}

	// return result
	if lineBreaksOnly { return height }
	return (height + self.getOpLineHeight()).QuantizeUp(vertQuant)
}

// returns the width quantized. doesn't include potential last \n in rune count
func (self *Renderer) helperMeasureLineLTR(iterator ltrStringIterator, text string) (ltrStringIterator, fract.Unit, int, rune) {
	var prevGlyphIndex sfnt.GlyphIndex
	var width fract.Unit
	var runeCount int

	horzQuant := fract.Unit(self.state.horzQuantization)
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 || codePoint == '\n' {
			return iterator, width.QuantizeUp(horzQuant), runeCount, codePoint
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

		// apply kerning unless no previous rune (line start)
		if runeCount > 0 {
			width += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
			width  = width.QuantizeUp(horzQuant)
		}

		// (here we would draw if we wanted to)

		// advance
		width += self.getOpAdvance(currGlyphIndex)
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
		runeCount += 1
	}
}

// returns the width quantized. doesn't include potential last \n in rune count
func (self *Renderer) helperMeasureLineReverseLTR(iterator ltrStringIterator, text string) (ltrStringIterator, fract.Unit, int, rune) {
	var prevGlyphIndex sfnt.GlyphIndex
	var width fract.Unit
	var runeCount int

	horzQuant := fract.Unit(self.state.horzQuantization)
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 || codePoint == '\n' {
			return iterator, -width, runeCount, codePoint
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

		// advance
		width -= self.getOpAdvance(currGlyphIndex)

		// apply kerning unless at line start
		if runeCount > 0 {
			width -= self.getOpKernBetween(currGlyphIndex, prevGlyphIndex)
		}
		
		// we need to quantize here inconditionally due to the previous advance
		width = width.QuantizeUp(horzQuant)
		
		// (here we would draw if we wanted to)

		// update tracking variables
		prevGlyphIndex = currGlyphIndex
		runeCount += 1
	}
}

// returns the width unquantized, without accounting for final wrapped spaces.
func (self *Renderer) helperMeasureWrapLineLTR(iterator ltrStringIterator, text string, widthLimit fract.Unit) (ltrStringIterator, fract.Unit, int, rune) {
	var x, lastSafeWidth fract.Unit
	var runeCount, lastSafeCount int
	var safeIterator ltrStringIterator
	var prevGlyphIndex sfnt.GlyphIndex

	horzQuant := fract.Unit(self.state.horzQuantization)
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 || codePoint == '\n' { return iterator, x, runeCount, codePoint }
		if codePoint == '\n' { return iterator, x, runeCount, codePoint }

		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

		// apply kerning unless at line start
		memoX := x
		if runeCount > 0 {
			x += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
			x = x.QuantizeUp(horzQuant)
		}

		// advance
		x += self.getOpAdvance(currGlyphIndex)

		// (here we would draw if we wanted to)

		// stop if outside wrapLimit
		runeCount += 1
		if codePoint == ' ' {
			lastSafeCount = runeCount
			lastSafeWidth = memoX
			safeIterator = iterator
		}
		if x > widthLimit && x.QuantizeUp(horzQuant) > widthLimit { // *
			// * the correctness of the quantized check is actually debatable, but it 
			//   does make for better consistency between measure and measureWithWrap,
			//   which seems more relevant in practical scenarios
			if lastSafeCount == 0 { // special case, show as much of first word as possible
				if runeCount == 1 {
					next := iterator.PeekNext(text)
					if next == -1 || next == '\n' { codePoint = next }
					if next == '\n' { iterator.Next(text) }
					return iterator, x, 1, codePoint
				} else {
					if codePoint != ' ' { iterator.Unroll(codePoint) }
					return iterator, memoX, runeCount - 1, codePoint
				}
			} else {
				return safeIterator, lastSafeWidth, lastSafeCount, ' '
			}
		}
		
		// update loop variables and continue
		prevGlyphIndex = currGlyphIndex
	}
}

// returns the width unquantized, without accounting for final wrapped spaces.
func (self *Renderer) helperMeasureWrapLineReverseLTR(iterator ltrStringIterator, text string, widthLimit fract.Unit) (ltrStringIterator, fract.Unit, int, rune) {
	var x, lastSafeWidth fract.Unit // values will be negative while looping
	var runeCount, lastSafeCount int
	var safeIterator ltrStringIterator
	var prevGlyphIndex sfnt.GlyphIndex

	horzQuant := fract.Unit(self.state.horzQuantization)
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 || codePoint == '\n' { return iterator, -x, runeCount, codePoint }
		if codePoint == '\n' { return iterator, -x, runeCount, codePoint }

		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

		// advance
		memoX := x
		x -= self.getOpAdvance(currGlyphIndex)

		// apply kerning unless at line start
		if runeCount > 0 {
			x -= self.getOpKernBetween(currGlyphIndex, prevGlyphIndex)
		}
		
		// we need to quantize here inconditionally due to the previous advance
		x = x.QuantizeUp(horzQuant)

		// (here we would draw if we wanted to)

		// stop if outside wrapLimit
		runeCount += 1
		if codePoint == ' ' {
			lastSafeCount = runeCount
			lastSafeWidth = memoX
			safeIterator = iterator
		}
		if x < -widthLimit && x.QuantizeUp(horzQuant) < -widthLimit {
			if lastSafeCount == 0 { // special case, show as much of first word as possible
				if runeCount == 1 {
					next := iterator.PeekNext(text)
					if next == -1 || next == '\n' { codePoint = next }
					if next == '\n' { iterator.Next(text) }
					return iterator, -x, 1, codePoint
				} else {
					if codePoint != ' ' { iterator.Unroll(codePoint) }
					return iterator, -memoX, runeCount - 1, codePoint
				}
			} else {
				return safeIterator, lastSafeWidth, lastSafeCount, ' '
			}
		}
		
		// update loop variables and continue
		prevGlyphIndex = currGlyphIndex
	}
}

// returns the width unquantized, without accounting for final wrapped spaces.
func (self *Renderer) helperMeasureWrapLineRTL(iterator rtlStringIterator, text string, widthLimit fract.Unit) (rtlStringIterator, fract.Unit, int, rune) {
	var x, lastSafeWidth fract.Unit // values will be negative while looping
	var runeCount, lastSafeCount int
	var safeIterator rtlStringIterator
	var prevGlyphIndex sfnt.GlyphIndex

	horzQuant := fract.Unit(self.state.horzQuantization)
	var memoIterator rtlStringIterator
	for {
		memoIterator = iterator
		codePoint := iterator.Next(text)
		if codePoint == -1 || codePoint == '\n' { return iterator, -x, runeCount, codePoint }
		if codePoint == '\n' { return iterator, -x, runeCount, codePoint }

		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

		// advance
		memoX := x
		x -= self.getOpAdvance(currGlyphIndex)

		// apply kerning unless at line start
		if runeCount > 0 {
			x -= self.getOpKernBetween(currGlyphIndex, prevGlyphIndex)
		}
		
		// we need to quantize here inconditionally due to the previous advance
		x = x.QuantizeUp(horzQuant)

		// (here we would draw if we wanted to)

		// stop if outside wrapLimit
		runeCount += 1
		if codePoint == ' ' {
			lastSafeCount = runeCount
			lastSafeWidth = memoX
			safeIterator = iterator
		}
		if x < -widthLimit && x.QuantizeUp(horzQuant) < -widthLimit {
			if lastSafeCount == 0 { // special case, show as much of first word as possible
				if runeCount == 1 {
					next := iterator.PeekNext(text)
					if next == -1 || next == '\n' { codePoint = next }
					if next == '\n' { iterator.Next(text) }
					return iterator, -x, 1, codePoint
				} else {
					if codePoint != ' ' { iterator = memoIterator }
					return iterator, -memoX, runeCount - 1, codePoint
				}
			} else {
				return safeIterator, lastSafeWidth, lastSafeCount, ' '
			}
		}
		
		// update loop variables and continue
		prevGlyphIndex = currGlyphIndex
	}
}

// returns the width unquantized, without accounting for final wrapped spaces.
func (self *Renderer) helperMeasureWrapLineReverseRTL(iterator rtlStringIterator, text string, widthLimit fract.Unit) (rtlStringIterator, fract.Unit, int, rune) {
	var x, lastSafeWidth fract.Unit
	var runeCount, lastSafeCount int
	var safeIterator rtlStringIterator
	var prevGlyphIndex sfnt.GlyphIndex

	horzQuant := fract.Unit(self.state.horzQuantization)
	var memoIterator rtlStringIterator
	for {
		memoIterator = iterator
		codePoint := iterator.Next(text)
		if codePoint == -1 || codePoint == '\n' { return iterator, x, runeCount, codePoint }
		if codePoint == '\n' { return iterator, x, runeCount, codePoint }

		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

		// apply kerning unless at line start
		memoX := x
		if runeCount > 0 {
			x += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
			x = x.QuantizeUp(horzQuant)
		}

		// advance
		x += self.getOpAdvance(currGlyphIndex)

		// (here we would draw if we wanted to)

		// stop if outside wrapLimit
		runeCount += 1
		if codePoint == ' ' {
			lastSafeCount = runeCount
			lastSafeWidth = memoX
			safeIterator = iterator
		}
		if x > widthLimit && x.QuantizeUp(horzQuant) > widthLimit { // *
			if lastSafeCount == 0 { // special case, show as much of first word as possible
				if runeCount == 1 {
					next := iterator.PeekNext(text)
					if next == -1 || next == '\n' { codePoint = next }
					if next == '\n' { iterator.Next(text) }
					return iterator, x, 1, codePoint
				} else {
					if codePoint != ' ' { iterator = memoIterator }
					return iterator, memoX, runeCount - 1, codePoint
				}
			} else {
				return safeIterator, lastSafeWidth, lastSafeCount, ' '
			}
		}
		
		// update loop variables and continue
		prevGlyphIndex = currGlyphIndex
	}
}
