package etxt

import (
	"github.com/tinne26/etxt/fract"
)

// Draws the given text with the current configuration (font, size, color,
// target, etc). The text drawing position is determined by the given pixel
// coordinates and the renderer's align (see [Renderer.SetAlign]() rules).
//
// Missing glyphs in the current font will cause the renderer to panic.
// See [RendererGlyph.GetRuneIndex]() for further advice if you need to
// make your system more robust.
func (self *Renderer) Draw(target Target, text string, x, y int) {
	self.fractDraw(target, text, fract.FromInt(x), fract.FromInt(y))
}

// x and y may be unquantized
func (self *Renderer) fractDraw(target Target, text string, x, y fract.Unit) {
	// preconditions
	if target == nil {
		panic("can't draw on nil Target")
	}
	if self.state.activeFont == nil {
		panic("can't draw text with nil font (tip: Renderer.SetFont())")
	}
	if self.state.fontSizer == nil {
		panic("can't draw with a nil sizer (tip: NewRenderer())")
	}
	if self.state.rasterizer == nil {
		panic("can't draw with a nil rasterizer (tip: NewRenderer())")
	}

	// return directly on superfluous invocations
	if text == "" {
		return
	}

	// adjust Y position
	horzQuant, vertQuant := self.fractGetQuantization()
	lineHeight := self.getOpLineHeight()
	vertAlign := self.state.align.Vert()
	switch vertAlign {
	case VertCenter:
		height := self.helperMeasureHeight(text)
		y = (y + self.getOpAscent() - (height >> 1)).QuantizeUp(vertQuant)
	case LastBaseline:
		height := self.helperMeasureHeight(text)
		qtLineHeight := lineHeight.QuantizeUp(vertQuant)
		if height >= qtLineHeight {
			height -= qtLineHeight
		}
		y = (y - height).QuantizeUp(vertQuant)
	case Bottom:
		height := self.helperMeasureHeight(text)
		y = (y + self.getOpAscent() - height).QuantizeUp(vertQuant)
	default:
		y = (y + self.getDistToBaselineFract(vertAlign)).QuantizeUp(vertQuant)
	}

	// Note: skipping text portions based on visibility can be a
	// problem when using custom draw and line break functions,
	// so I'm temporarily suspending the optimization

	// skip non-visible portions of the text in the target
	// (ascent and descent would be enough for most properly
	//  made fonts, but using line height is safer)
	// minBaselineY := fract.FromInt(bounds.Min.Y) - lineHeight
	// maxBaselineY := fract.FromInt(bounds.Max.Y) + lineHeight
	// var lineBreakNth int = -1
	// if y < minBaselineY {
	// 	var iSkip int
	// 	for i, codePoint := range text {
	// 		if codePoint == '\n' {
	// 			lineBreakNth = maxInt(1, lineBreakNth+1)
	// 			lineBreakNth += 1
	// 			y += self.getOpLineAdvance(lineBreakNth)
	// 			y = y.QuantizeUp(vertQuant)
	// 			iSkip = i + 1
	// 			if y >= minBaselineY {
	// 				break
	// 			}
	// 		}
	// 	}
	// 	text = text[iSkip:]
	// 	if text == "" {
	// 		return
	// 	}
	// }

	// subdelegate to relevant draw function
	switch self.state.align.Horz() {
	case Left:
		if self.state.textDirection == LeftToRight {
			self.fractDrawLeftLTR(target, text, x.QuantizeUp(horzQuant), y)
		} else {
			self.fractDrawLeftRTL(target, text, x.QuantizeUp(horzQuant), y)
		}
	case Right:
		if self.state.textDirection == LeftToRight {
			self.fractDrawRightLTR(target, text, x.QuantizeUp(horzQuant), y)
		} else {
			self.fractDrawRightRTL(target, text, x.QuantizeUp(horzQuant), y)
		}
	case HorzCenter:
		if self.state.textDirection == LeftToRight {
			self.fractDrawCenterLTR(target, text, x, y)
		} else {
			self.fractDrawCenterRTL(target, text, x, y)
		}
	default:
		panic(self.state.align.Horz())
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawLeftLTR(target Target, text string, x, y fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = -1
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator ltrStringIterator
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 {
			break
		}
		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if self.lineChangeFn != nil {
				self.lineChangeFn(iv.lineChangeDetails)
			}
		} else {
			position, iv = self.drawRuneLTR(target, position, codePoint, iv)
		}
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawLeftRTL(target Target, text string, x, y fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = -1
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator rtlStringIterator
	iterator.Init(text)
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 {
			break
		}
		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if self.lineChangeFn != nil {
				self.lineChangeFn(iv.lineChangeDetails)
			}
		} else {
			position, iv = self.drawRuneLTR(target, position, codePoint, iv)
		}
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawRightLTR(target Target, text string, x, y fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = -1
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator rtlStringIterator
	iterator.Init(text)
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 {
			break
		}
		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if self.lineChangeFn != nil {
				self.lineChangeFn(iv.lineChangeDetails)
			}
		} else {
			position, iv = self.drawRuneRTL(target, position, codePoint, iv)
		}
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawRightRTL(target Target, text string, x, y fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = -1
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator ltrStringIterator
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 {
			break
		}
		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if self.lineChangeFn != nil {
				self.lineChangeFn(iv.lineChangeDetails)
			}
		} else {
			position, iv = self.drawRuneRTL(target, position, codePoint, iv)
		}
	}
}

// Precondition: y is already quantized, x is not (for better precision).
func (self *Renderer) fractDrawCenterLTR(target Target, text string, x, y fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = -1
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator ltrStringIterator
	for {
		codePoint := iterator.PeekNext(text)
		if codePoint == -1 {
			break
		} // we are done
		if codePoint == '\n' { // deal with line breaks
			_ = iterator.Next(text) // consume line break
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if self.lineChangeFn != nil {
				self.lineChangeFn(iv.lineChangeDetails)
			}
			continue
		}

		_, lineWidth, runeCount, _ := self.helperMeasureLineLTR(iterator, text)
		position.X = x - (lineWidth >> 1)
		_, iv, iterator = self.helperDrawLineLTR(target, position, iv, iterator, text, runeCount)
	}
}

// Precondition: y is already quantized, x is not (for better precision).
func (self *Renderer) fractDrawCenterRTL(target Target, text string, x, y fract.Unit) {
	// There are multiple approaches here:
	// - iterate text from left to right, but measure and draw in reverse
	// - iterate from right to left, but measure and draw normally
	// The first is slightly nicer due to ltr iterator being simpler

	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = -1
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator ltrStringIterator
	for {
		codePoint := iterator.PeekNext(text)
		if codePoint == -1 {
			break
		} // we are done
		if codePoint == '\n' { // deal with line breaks
			_ = iterator.Next(text) // consume line break
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if self.lineChangeFn != nil {
				self.lineChangeFn(iv.lineChangeDetails)
			}
			continue
		}

		_, lineWidth, runeCount, _ := self.helperMeasureLineReverseLTR(iterator, text)
		position.X = x + (lineWidth >> 1)
		_, iv, iterator = self.helperDrawLineReverseLTR(target, position, iv, iterator, text, runeCount)
	}
}
