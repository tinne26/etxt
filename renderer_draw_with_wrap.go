package etxt

import "github.com/tinne26/etxt/fract"

// Same as [Renderer.Draw](), but using a width limit for line wrapping.
// The algorithm is a trivial greedy algorithm that only considers spaces
// as line wrapping candidates.
//
// The widthLimit must be given in real units, not logical ones.
// This means that unlike text sizes, the widthLimit won't be internally
// multiplied by the renderer's scale factor.
func (self *Renderer) DrawWithWrap(target Target, text string, x, y, widthLimit int) {
	if widthLimit > fract.MaxInt { panic("widthLimit too big, must be <= fract.MaxInt") }
	self.fractDrawWithWrap(target, text, fract.FromInt(x), fract.FromInt(y), fract.FromInt(widthLimit))
}

// x and y are assumed to be unquantized
func (self *Renderer) fractDrawWithWrap(target Target, text string, x, y fract.Unit, widthLimit fract.Unit) {
	// preconditions
	if target == nil { panic("can't draw on nil Target") }
	if self.state.activeFont == nil { panic("can't draw text with nil font (tip: Renderer.SetFont())") }
	if self.state.fontSizer  == nil { panic("can't draw with a nil sizer (tip: NewRenderer())") }
	if self.state.rasterizer == nil { panic("can't draw with a nil rasterizer (tip: NewRenderer())") }

	// return directly on superfluous invocations
	if text == "" { return }
	bounds := target.Bounds()
	if bounds.Empty() { return }
	
	// adjust Y position
	horzQuant, vertQuant := self.fractGetQuantization()
	lineHeight := self.getOpLineHeight()
	vertAlign := self.state.align.Vert()
	switch vertAlign {
	case Top:
		y = (y + self.getOpAscent()).QuantizeUp(vertQuant)
	case CapLine:
		capHeight := self.getSlowOpCapHeight()
		y = (y + capHeight).QuantizeUp(vertQuant)
	case Midline:
		xheight := self.getSlowOpXHeight()
		y = (y + xheight).QuantizeUp(vertQuant)
	case VertCenter:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		y = (y + self.getOpAscent() - (height >> 1)).QuantizeUp(vertQuant)
	case Baseline:
		y = y.QuantizeUp(vertQuant)
	case LastBaseline:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		qtLineHeight := lineHeight.QuantizeUp(vertQuant)
		if height >= qtLineHeight { height -= qtLineHeight }
		y = (y - height).QuantizeUp(vertQuant)
	case Bottom:
		height := self.helperMeasureHeight(text)
		y = (y + self.getOpAscent() - height).QuantizeUp(vertQuant)
	default:
		panic(vertAlign)
	}

	// skip non-visible portions of the text in the target
	minBaselineY := fract.FromInt(bounds.Min.Y) - lineHeight
	maxBaselineY := fract.FromInt(bounds.Max.Y) + lineHeight
	var lineBreakNth int
	text, lineBreakNth = self.trimNonVisibleWithWrap(text, widthLimit, y, minBaselineY)
	if text == "" { return }

	// subdelegate to the relevant function
	x = x.QuantizeUp(horzQuant)
	switch self.state.align.Horz() {
	case Left:
		if self.state.textDirection == LeftToRight {
			self.fractDrawWithWrapLeftLTR(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
		} else {
			self.fractDrawWithWrapLeftRTL(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
		}
	case Right:
		if self.state.textDirection == LeftToRight {
			self.fractDrawWithWrapRightLTR(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
		} else {
			self.fractDrawWithWrapRightRTL(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
		}
	case HorzCenter:
		if self.state.textDirection == LeftToRight {
			self.fractDrawWithWrapCenterLTR(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
		} else {
			self.fractDrawWithWrapCenterRTL(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
		}
	default:
		panic(self.state.align.Horz())
	}
}

// y must be quantized, min baseline not
func (self *Renderer) trimNonVisibleWithWrap(text string, widthLimit, y, minBaselineY fract.Unit) (string, int) {
	if y >= minBaselineY { return text, -1 }
	if self.state.textDirection == LeftToRight {
		return self.trimNonVisibleWithWrapLTR(text, widthLimit, y, minBaselineY)
	} else { // assume textDirection == RightToLeft
		return self.trimNonVisibleWithWrapRTL(text, widthLimit, y, minBaselineY)
	}
}

func (self *Renderer) trimNonVisibleWithWrapLTR(text string, widthLimit, y, minBaselineY fract.Unit) (string, int) {
	var lineBreakNth int = -1
	var iterator ltrStringIterator
	var lastRune rune
	for {
		iterator, _, _, lastRune = self.helperMeasureWrapLineLTR(iterator, text, widthLimit)
		if lastRune == '\n' {
			lineBreakNth = maxInt(1, lineBreakNth + 1)
		} else {
			lineBreakNth = 0
		}
		if lastRune == -1 { return "", lineBreakNth }
		y += self.getOpLineAdvance(lineBreakNth)
		if y >= minBaselineY { break }
	}
	return iterator.StringLeft(text), lineBreakNth
}

func (self *Renderer) trimNonVisibleWithWrapRTL(text string, widthLimit, y, minBaselineY fract.Unit) (string, int) {
	var lineBreakNth int = -1
	var iterator ltrStringIterator
	var lastRune rune
	for {
		iterator, _, _, lastRune = self.helperMeasureWrapLineReverseLTR(iterator, text, widthLimit)
		if lastRune == '\n' {
			lineBreakNth = maxInt(1, lineBreakNth + 1)
		} else {
			lineBreakNth = 0
		}
		if lastRune == -1 { return "", lineBreakNth }
		y += self.getOpLineAdvance(lineBreakNth)
		if y >= minBaselineY { break }
	}
	return iterator.StringLeft(text), lineBreakNth
}

// Precondition: x and y are quantized.
func (self *Renderer) fractDrawWithWrapLeftLTR(target Target, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = lineBreakNth
	for { // for each wrap line
		// Notice: this approach is not optimal. One could write a helperDrawWrapLineLTR 
		//         directly. Same goes for the other 3 horz align + text dir variants.
		_, _, runeCount, lastRune := self.helperMeasureWrapLineLTR(iterator, text, widthLimit)
		if runeCount > 0 {
			position, iv, iterator = self.helperDrawLineLTR(target, position, iv, iterator, text, runeCount)
		}

		// stop here or advance line
		if lastRune == -1 { break }
		if lastRune == '\n' { _ = iterator.Next(text) }
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, startX, iv.lineBreakNth)
		if position.Y > maxY { break }
	}
}

// Precondition: x and y are quantized.
func (self *Renderer) fractDrawWithWrapLeftRTL(target Target, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	// set up traversal variables
	var iterator rtlStringIterator
	iterator.Init(text)
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = lineBreakNth
	for { // for each wrap line
		_, _, runeCount, lastRune := self.helperMeasureWrapLineReverseRTL(iterator, text, widthLimit)
		if runeCount > 0 {
			position, iv, iterator = self.helperDrawLineReverseRTL(target, position, iv, iterator, text, runeCount)
		}
		
		// stop here or advance line
		if lastRune == -1 { break }
		if lastRune == '\n' { _ = iterator.Next(text) }
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, startX, iv.lineBreakNth)
		if position.Y > maxY { break }
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawWithWrapRightLTR(target Target, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {	
	// set up traversal variables
	var iterator rtlStringIterator
	iterator.Init(text)
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = lineBreakNth
	for { // for each wrap line
		_, _, runeCount, lastRune := self.helperMeasureWrapLineRTL(iterator, text, widthLimit)
		if runeCount > 0 {
			position, iv, iterator = self.helperDrawLineRTL(target, position, iv, iterator, text, runeCount)
		}

		// stop here or advance line
		if lastRune == -1 { break }
		if lastRune == '\n' { _ = iterator.Next(text) }
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, startX, iv.lineBreakNth)
		if position.Y > maxY { break }
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawWithWrapRightRTL(target Target, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = lineBreakNth
	for { // for each wrap line
		_, _, runeCount, lastRune := self.helperMeasureWrapLineReverseLTR(iterator, text, widthLimit)
		if runeCount > 0 {
			position, iv, iterator = self.helperDrawLineReverseLTR(target, position, iv, iterator, text, runeCount)
		}

		// stop here or advance line
		if lastRune == -1 { break }
		if lastRune == '\n' { _ = iterator.Next(text) }
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, startX, iv.lineBreakNth)
		if position.Y > maxY { break }
	}
}

// Precondition: y is already quantized, x is not (for better precision).
func (self *Renderer) fractDrawWithWrapCenterLTR(target Target, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = lineBreakNth
	for { // for each wrap line
		_, lineWidth, runeCount, lastRune := self.helperMeasureWrapLineLTR(iterator, text, widthLimit)
		if runeCount > 0 {
			position.X = x - (lineWidth >> 1)
			_, iv, iterator = self.helperDrawLineLTR(target, position, iv, iterator, text, runeCount)
		}

		// stop here or advance line
		if lastRune == -1 { break }
		if lastRune == '\n' { _ = iterator.Next(text) }
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, 0, iv.lineBreakNth)
		if position.Y > maxY { break }
	}
}

// Precondition: y is already quantized, x is not (for better precision).
func (self *Renderer) fractDrawWithWrapCenterRTL(target Target, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = lineBreakNth
	for { // for each wrap line
		_, lineWidth, runeCount, lastRune := self.helperMeasureWrapLineReverseLTR(iterator, text, widthLimit)
		if runeCount > 0 {
			position.X = x + (lineWidth >> 1)
			position, iv, iterator = self.helperDrawLineReverseLTR(target, position, iv, iterator, text, runeCount)
		}

		// stop here or advance line
		if lastRune == -1 { break }
		if lastRune == '\n' { _ = iterator.Next(text) }
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, 0, iv.lineBreakNth)
		if position.Y > maxY { break }
	}
}
