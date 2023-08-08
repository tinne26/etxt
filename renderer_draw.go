package etxt

import "github.com/tinne26/etxt/fract"

// TODO: verify that quantization rounding is ok in both directions.

// Draws the given text with the current configuration (font, size, color,
// target, etc). The position at which the text will be drawn depends on
// the given pixel coordinates and the renderer's align (see
// [Renderer.SetAlign]() rules).
//
// Missing glyphs in the current font will cause the renderer to panic.
// See [RendererGlyph.GetRuneIndex]() for further advice if you need to
// make your system more robust.
func (self *Renderer) Draw(target TargetImage, text string, x, y int) {
	self.fractDraw(target, text, fract.FromInt(x), fract.FromInt(y))
}

// x and y may be unquantized
func (self *Renderer) fractDraw(target TargetImage, text string, x, y fract.Unit) {
	// return directly on superfluous invocations
	if text == "" { return }
	bounds := target.Bounds()
	if bounds.Empty() { return }
	
	// preconditions
	if target == nil { panic("can't draw on nil TargetImage") }
	if self.state.activeFont == nil { panic("can't draw text with nil font (tip: Renderer.SetFont())") }
	if self.state.fontSizer  == nil { panic("can't draw with a nil sizer (tip: NewRenderer())") }
	if self.state.rasterizer == nil { panic("can't draw with a nil rasterizer (tip: NewRenderer())") }
	
	// adjust Y position
	horzQuant, vertQuant := self.fractGetQuantization()
	ascent := self.getOpAscent()
	lineHeight := self.getOpLineHeight()
	vertAlign := self.state.align.Vert()
	switch vertAlign {
	case Top:
		y = (y + ascent).QuantizeUp(vertQuant)
	case Midline:
		xheight := self.getSlowOpXHeight()
		y = (y + xheight).QuantizeUp(vertQuant)
	case VertCenter:
		height := self.fractMeasureHeight(text)
		y = (y + ascent - (height >> 1)).QuantizeUp(vertQuant)
	case Baseline:
		y = y.QuantizeUp(vertQuant)
	case LastBaseline, LastMidline:
		height := self.fractMeasureHeight(text)
		qtLineHeight := lineHeight.QuantizeUp(vertQuant)
		if height >= qtLineHeight { height -= qtLineHeight }
		y = (y - height).QuantizeUp(vertQuant)
		if vertAlign == LastMidline {
			y += self.getSlowOpXHeight()
		}
	case Bottom:
		height := self.fractMeasureHeight(text)
		y = (y + ascent - height).QuantizeUp(vertQuant)
	default:
		panic(vertAlign)
	}

	// skip non-visible portions of the text in the target
	// (ascent and descent would be enough for most properly
	//  made fonts, but using line height is safer)
	minBaselineY := fract.FromInt(bounds.Min.Y) - lineHeight
	maxBaselineY := fract.FromInt(bounds.Max.Y) + lineHeight
	var lineBreakNth int = -1
	if y < minBaselineY {
		for i, codePoint := range text {
			if codePoint == '\n' {
				lineBreakNth = maxInt(1, lineBreakNth + 1)
				lineBreakNth += 1
				y += self.getOpLineAdvance(lineBreakNth)
				y  = y.QuantizeUp(vertQuant)
				if y >= minBaselineY {
					text = text[i + 1 : ]
					break
				}
			}
		}
	}
	if text == "" { return }

	// subdelegate to relevant draw function
	ltr := (self.state.textDirection == LeftToRight)
	switch self.state.align.Horz() {
	case Left:
		if ltr {
			self.fractDrawLeftLTR(target, text, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
		} else {
			self.fractDrawLeftRTL(target, text, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
		}
	case Right:
		if ltr {
			self.fractDrawRightLTR(target, text, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
		} else {
			self.fractDrawRightRTL(target, text, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
		}
	case HorzCenter:
		if ltr {
			self.fractDrawCenterLTR(target, text, lineBreakNth, x, y, maxBaselineY)
		} else {
			self.fractDrawCenterRTL(target, text, lineBreakNth, x, y, maxBaselineY)
		}
	default:
		panic(self.state.align.Horz())
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawLeftLTR(target TargetImage, text string, lineBreakNth int, x, y, maxY fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator ltrStringIterator
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 { break }
		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if position.Y > maxY { break }
		} else {
			position, iv = self.drawGlyphLTR(target, position, codePoint, iv)
		}
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawLeftRTL(target TargetImage, text string, lineBreakNth int, x, y, maxY fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator rtlStringIterator
	iterator.Init(text)
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 { break }
		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if position.Y > maxY { break }
		} else {
			position, iv = self.drawGlyphLTR(target, position, codePoint, iv)
		}
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawRightLTR(target TargetImage, text string, lineBreakNth int, x, y, maxY fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator rtlStringIterator
	iterator.Init(text)
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 { break }
		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if position.Y > maxY { break }
		} else {
			position, iv = self.drawGlyphRTL(target, position, codePoint, iv)
		}
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawRightRTL(target TargetImage, text string, lineBreakNth int, x, y, maxY fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator ltrStringIterator
	for {
		codePoint := iterator.Next(text)
		if codePoint == -1 { break }
		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if position.Y > maxY { break }
		} else {
			position, iv = self.drawGlyphRTL(target, position, codePoint, iv)
		}
	}
}

// Precondition: y is already quantized, x is not (for better precision).
func (self *Renderer) fractDrawCenterLTR(target TargetImage, text string, lineBreakNth int, x, y, maxY fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = lineBreakNth

	var iterator ltrStringIterator
	for {
		codePoint := iterator.PeekNext(text)
		if codePoint == -1 { break } // we are done
		if codePoint == '\n' { // deal with line breaks
			_ = iterator.Next(text) // consume line break
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if position.Y > maxY { break }
			continue
		}
		
		memoIter := iterator
		width, runeCount := self.measureLineLTR(iterator, text)
		if runeCount > 0 {
			position.X = x - (width >> 1)
			iv.prevFractX = position.X.FractShift()
			_, iv, iterator = self.drawLineLTR(target, position, iv, memoIter, text, runeCount)
		}
	}
}

// Precondition: y is already quantized, x is not (for better precision).
func (self *Renderer) fractDrawCenterRTL(target TargetImage, text string, lineBreakNth int, x, y, maxY fract.Unit) {
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = lineBreakNth

	var iterator ltrStringIterator
	for {
		codePoint := iterator.PeekNext(text)
		if codePoint == -1 { break } // we are done
		if codePoint == '\n' { // deal with line breaks
			_ = iterator.Next(text) // consume line break
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if position.Y > maxY { break }
			continue
		}
		
		memoIter := iterator
		width, runeCount := self.measureLineRTL(iterator, text)
		if runeCount > 0 {
			position.X = x + (width >> 1)
			iv.prevFractX = position.X.FractShift()
			_, iv, iterator = self.drawLineRTL(target, position, iv, memoIter, text, runeCount)
		}
	}
}
