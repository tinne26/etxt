package etxt

import (
	"github.com/tinne26/etxt/fract"
)

// Same as [Renderer.Draw](), but using a width limit for line wrapping.
// The algorithm is a trivial greedy algorithm that only considers spaces
// as line wrapping candidates.
//
// The widthLimit must be given in real pixels, not logical units.
// This means that unlike text sizes, the widthLimit won't be internally
// multiplied by the renderer's scale factor.
func (self *Renderer) DrawWithWrap(target Target, text string, x, y, widthLimit int) {
	if widthLimit > fract.MaxInt {
		panic("widthLimit too big, must be <= fract.MaxInt")
	}
	self.fractDrawWithWrap(target, text, fract.FromInt(x), fract.FromInt(y), fract.FromInt(widthLimit))
}

// x and y are assumed to be unquantized
func (self *Renderer) fractDrawWithWrap(target Target, text string, x, y fract.Unit, widthLimit fract.Unit) {
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
	bounds := target.Bounds()
	if bounds.Empty() {
		return
	}

	// adjust Y position
	horzQuant, vertQuant := self.fractGetQuantization()
	lineHeight := self.getOpLineHeight()
	vertAlign := self.state.align.Vert()
	switch vertAlign {
	case VertCenter:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		y = (y + self.getOpAscent() - (height >> 1)).QuantizeUp(vertQuant)
	case LastBaseline:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		qtLineHeight := lineHeight.QuantizeUp(vertQuant)
		if height >= qtLineHeight {
			height -= qtLineHeight
		}
		y = (y - height).QuantizeUp(vertQuant)
	case Bottom:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		y = (y + self.getOpAscent() - height).QuantizeUp(vertQuant)
	default:
		y = (y + self.getDistToBaselineFract(vertAlign)).QuantizeUp(vertQuant)
	}

	// Note: skipping text portions based on visibility can be a
	// problem when using custom draw and line break functions,
	// so I'm temporarily suspending the optimization

	// skip non-visible portions of the text in the target
	// minBaselineY := fract.FromInt(bounds.Min.Y) - lineHeight
	// maxBaselineY := fract.FromInt(bounds.Max.Y) + lineHeight
	// var lineBreakNth int
	// text, lineBreakNth = self.trimNonVisibleWithWrap(text, widthLimit, y, minBaselineY)
	// if text == "" {
	// 	return
	// }

	// subdelegate to the relevant function
	x = x.QuantizeUp(horzQuant)
	switch self.state.align.Horz() {
	case Left:
		if self.state.textDirection == LeftToRight {
			self.fractDrawWithWrapLeftLTR(target, text, x, y, widthLimit)
		} else {
			self.fractDrawWithWrapLeftRTL(target, text, x, y, widthLimit)
		}
	case Right:
		if self.state.textDirection == LeftToRight {
			self.fractDrawWithWrapRightLTR(target, text, x, y, widthLimit)
		} else {
			self.fractDrawWithWrapRightRTL(target, text, x, y, widthLimit)
		}
	case HorzCenter:
		if self.state.textDirection == LeftToRight {
			self.fractDrawWithWrapCenterLTR(target, text, x, y, widthLimit)
		} else {
			self.fractDrawWithWrapCenterRTL(target, text, x, y, widthLimit)
		}
	default:
		panic(self.state.align.Horz())
	}
}

// y must be quantized, min baseline not
func (self *Renderer) trimNonVisibleWithWrap(text string, widthLimit, y, minBaselineY fract.Unit) (string, int) {
	if y >= minBaselineY {
		return text, -1
	}
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
			lineBreakNth = maxInt(1, lineBreakNth+1)
		} else {
			lineBreakNth = 0
		}
		if lastRune == -1 {
			return "", lineBreakNth
		}
		y += self.getOpLineAdvance(lineBreakNth)
		if y >= minBaselineY {
			break
		}
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
			lineBreakNth = maxInt(1, lineBreakNth+1)
		} else {
			lineBreakNth = 0
		}
		if lastRune == -1 {
			return "", lineBreakNth
		}
		y += self.getOpLineAdvance(lineBreakNth)
		if y >= minBaselineY {
			break
		}
	}
	return iterator.StringLeft(text), lineBreakNth
}

// Precondition: x and y are quantized.
func (self *Renderer) fractDrawWithWrapLeftLTR(target Target, text string, x, y, widthLimit fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = -1
	for { // for each wrap line
		// Notice: this approach is not optimal. One could write a helperDrawWrapLineLTR
		//         directly. Same goes for the other 3 horz align + text dir variants.
		_, _, runeCount, lastRune := self.helperMeasureWrapLineLTR(iterator, text, widthLimit)
		iv.updateLineChangeFromWrapMeasure(runeCount, lastRune)
		position, iv, iterator = self.helperDrawLineLTR(target, position, iv, iterator, text, runeCount-iv.numElisions())
		if iv.lineChangeDetails.ElidedSpace {
			if iterator.Next(text) != ' ' {
				panic("broken code")
			}
		}

		// stop here or advance line
		if lastRune == -1 {
			break
		}
		if lastRune == '\n' {
			_ = iterator.Next(text)
		}
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, startX, iv.lineBreakNth)
		if self.lineChangeFn != nil {
			self.lineChangeFn(iv.lineChangeDetails)
		}
	}
}

// Precondition: x and y are quantized.
func (self *Renderer) fractDrawWithWrapLeftRTL(target Target, text string, x, y, widthLimit fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = -1
	for { // for each wrap line
		_, lineWidth, runeCount, lastRune := self.helperMeasureWrapLineReverseLTR(iterator, text, widthLimit)
		iv.updateLineChangeFromWrapMeasure(runeCount, lastRune)
		position.X = x + lineWidth
		position, iv, iterator = self.helperDrawLineReverseLTR(target, position, iv, iterator, text, runeCount-iv.numElisions())
		if iv.lineChangeDetails.ElidedSpace {
			if iterator.Next(text) != ' ' {
				panic("broken code")
			}
		}

		// stop here or advance line
		if lastRune == -1 {
			break
		}
		if lastRune == '\n' {
			_ = iterator.Next(text)
		}
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, startX, iv.lineBreakNth)
		if self.lineChangeFn != nil {
			self.lineChangeFn(iv.lineChangeDetails)
		}
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawWithWrapRightLTR(target Target, text string, x, y, widthLimit fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = -1
	for { // for each wrap line
		_, lineWidth, runeCount, lastRune := self.helperMeasureWrapLineLTR(iterator, text, widthLimit)
		iv.updateLineChangeFromWrapMeasure(runeCount, lastRune)
		position.X = startX - lineWidth
		position, iv, iterator = self.helperDrawLineLTR(target, position, iv, iterator, text, runeCount-iv.numElisions())
		if iv.lineChangeDetails.ElidedSpace {
			if iterator.Next(text) != ' ' {
				panic("broken code")
			}
		}

		// stop here or advance line
		if lastRune == -1 {
			break
		}
		if lastRune == '\n' {
			_ = iterator.Next(text)
		}
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, startX, iv.lineBreakNth)
		if self.lineChangeFn != nil {
			self.lineChangeFn(iv.lineChangeDetails)
		}
	}
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawWithWrapRightRTL(target Target, text string, x, y, widthLimit fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = -1
	for { // for each wrap line
		_, _, runeCount, lastRune := self.helperMeasureWrapLineReverseLTR(iterator, text, widthLimit)
		iv.updateLineChangeFromWrapMeasure(runeCount, lastRune)
		position, iv, iterator = self.helperDrawLineReverseLTR(target, position, iv, iterator, text, runeCount-iv.numElisions())
		if iv.lineChangeDetails.ElidedSpace {
			if iterator.Next(text) != ' ' {
				panic("broken code")
			}
		}

		// stop here or advance line
		if lastRune == -1 {
			break
		}
		if lastRune == '\n' {
			_ = iterator.Next(text)
		}
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, startX, iv.lineBreakNth)
		if self.lineChangeFn != nil {
			self.lineChangeFn(iv.lineChangeDetails)
		}
	}
}

// Precondition: y is already quantized, x is not (for better precision).
func (self *Renderer) fractDrawWithWrapCenterLTR(target Target, text string, x, y, widthLimit fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = -1
	for { // for each wrap line
		_, lineWidth, runeCount, lastRune := self.helperMeasureWrapLineLTR(iterator, text, widthLimit)
		iv.updateLineChangeFromWrapMeasure(runeCount, lastRune)
		position.X = x - (lineWidth >> 1)
		_, iv, iterator = self.helperDrawLineLTR(target, position, iv, iterator, text, runeCount-iv.numElisions())
		if iv.lineChangeDetails.ElidedSpace {
			if iterator.Next(text) != ' ' {
				panic("broken code")
			}
		}

		// stop here or advance line
		if lastRune == -1 {
			break
		}
		if lastRune == '\n' {
			_ = iterator.Next(text)
		}

		iv.increaseLineBreakNth()
		position = self.advanceLine(position, 0, iv.lineBreakNth)
		if self.lineChangeFn != nil {
			self.lineChangeFn(iv.lineChangeDetails)
		}
	}
}

// Precondition: y is already quantized, x is not (for better precision).
func (self *Renderer) fractDrawWithWrapCenterRTL(target Target, text string, x, y, widthLimit fract.Unit) {
	// set up traversal variables
	var iterator ltrStringIterator
	var position fract.Point = fract.UnitsToPoint(x, y)
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iv drawInternalValues
	iv.prevFractX = x.FractShift()
	iv.lineBreakNth = -1
	for { // for each wrap line
		_, lineWidth, runeCount, lastRune := self.helperMeasureWrapLineReverseLTR(iterator, text, widthLimit)
		iv.updateLineChangeFromWrapMeasure(runeCount, lastRune)
		position.X = x + (lineWidth >> 1)
		position, iv, iterator = self.helperDrawLineReverseLTR(target, position, iv, iterator, text, runeCount-iv.numElisions())
		if iv.lineChangeDetails.ElidedSpace {
			if iterator.Next(text) != ' ' {
				panic("broken code")
			}
		}

		// stop here or advance line
		if lastRune == -1 {
			break
		}
		if lastRune == '\n' {
			_ = iterator.Next(text)
		}
		iv.increaseLineBreakNth()
		position = self.advanceLine(position, 0, iv.lineBreakNth)
		if self.lineChangeFn != nil {
			self.lineChangeFn(iv.lineChangeDetails)
		}
	}
}
