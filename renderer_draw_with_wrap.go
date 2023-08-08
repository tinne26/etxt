package etxt

import "unicode/utf8"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// TODO: clearly detail if values are quantized or not from the start. because in some functions
//       I'm passing unquantized values. and then assumming they are quantized. etc. it's a mess,
//       it needs to be specified clearly.

// Same as [Renderer.Draw](), but using a width limit for line wrapping.
// The line wrapping algorithm is a trivial greedy algorithm using
// spaces as the only line breaking points.
//
// The widthLimit must must be given in real units, not logical ones.
// This means that unlike text sizes, the widthLimit won't be internally
// multiplied by the renderer's scale factor.
func (self *Renderer) DrawWithWrap(target TargetImage, text string, x, y, widthLimit int) {
	if widthLimit > fract.MaxInt { panic("widthLimit too big, must be <= fract.MaxInt") }
	self.fractDrawWithWrap(target, text, fract.FromInt(x), fract.FromInt(y), fract.FromInt(widthLimit))
}

// x and y are assumed to be unquantized
func (self *Renderer) fractDrawWithWrap(target TargetImage, text string, x, y fract.Unit, widthLimit fract.Unit) {
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
	ascent := self.state.fontSizer.Ascent(self.state.activeFont, &self.buffer, self.state.scaledSize)
	vertAlign := self.state.align.Vert()
	switch vertAlign {
	case Top:
		y = (y + ascent).QuantizeUp(vertQuant)
	case Midline:
		xheight := self.getSlowOpXHeight()
		y = (y + xheight).QuantizeUp(vertQuant)
	case VertCenter:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		y = (y + ascent - (height >> 1)).QuantizeUp(vertQuant)
	case Baseline:
		y = y.QuantizeUp(vertQuant)
	case LastBaseline, LastMidline:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		lineHeight := self.getOpLineHeight().QuantizeUp(vertQuant)
		if height >= lineHeight { height -= lineHeight }
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
	minBaselineY := fract.FromInt(bounds.Min.Y) - ascent
	maxBaselineY := fract.FromInt(bounds.Max.Y) + ascent
	var lineBreakNth int = -1
	if y < minBaselineY {
		byteCount := 0
		iterator := newStrIterator(text, false)
		for {
			_, runeCount, lineBreak, eot := self.measureWrapLine(iterator, widthLimit)
			for runeCount > 0 {
				codePoint := iterator.Next()
				byteCount += utf8.RuneLen(codePoint)
			}
			if lineBreak {
				_ = iterator.Next()
				byteCount += 1
				lineBreakNth = maxInt(1, lineBreakNth + 1)
			} else {
				lineBreakNth = 0
			}
			if eot { return }
			y += self.getOpLineAdvance(lineBreakNth)
			if y >= minBaselineY { break }
		}
		text = text[byteCount : ]
	}
	if text == "" { return }
	

	// subdelegate to the relevant function
	x = x.QuantizeUp(horzQuant)
	switch self.state.align.Horz() {
	case Left:
		self.fractDrawWithWrapLeft(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
	case Right:
		self.fractDrawWithWrapRight(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
	case HorzCenter:
		self.fractDrawWithWrapCenter(target, text, lineBreakNth, x, y, widthLimit, maxBaselineY)
	default:
		panic(self.state.align.Horz())
	}
}

// Precondition: x and y are quantized.
func (self *Renderer) fractDrawWithWrapLeft(target TargetImage, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	// Technical notes:
	// - I'm never skipping spaces (unlike when measuring) in case a custom
	//   glyph draw func is being used and it expects consistency. Of course,
	//   other parts of the pipeline may break this consistency anyway (like
	//   skipping non-visible y regions). Hard to determine what's the best
	//   way to go in the situation.
	// - If end of text coincides with an overflowing first-glyph-of-line,
	//   we account for that and advance the line, even if currently we
	//   aren't returning or storing the final position. We might be
	//   interested in this behavior at some point in the future.
	
	// create string iterator
	iterator := newStrIterator(text, false)

	// set up traversal variables
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	for { // for each wrap line
		width, runeCount, lineBreak, eot := self.measureWrapLine(iterator, widthLimit)
		if self.state.textDirection == RightToLeft {
			// already properly quantized unless using non-equidistant quantizations,
			// but in that case other minor things will be messed up anyway
			position.X = (startX + width).QuantizeUp(fract.Unit(self.state.horzQuantization))
		}
		position, iterator = self.drawWrapLine(target, position, iterator, runeCount, lineBreak)

		// advance line unless on last line without line break
		if lineBreak || !eot {
			lineBreakNth = maxInt(1, lineBreakNth + 1)
			position = self.advanceLine(position, startX, lineBreakNth)
			if position.Y > maxY { break }
		}

		if eot { break }
	}
}

func (self *Renderer) fractDrawWithWrapRight(target TargetImage, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {	
	// create string iterator
	iterator := newStrIterator(text, false)

	// set up traversal variables
	var position fract.Point = fract.UnitsToPoint(x, y)
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	for { // for each wrap line
		width, runeCount, lineBreak, eot := self.measureWrapLine(iterator, widthLimit)
		if self.state.textDirection == LeftToRight {
			// already properly quantized unless using non-equidistant quantizations,
			// but in that case other minor things will be messed up anyway
			position.X = (startX + width).QuantizeUp(fract.Unit(self.state.horzQuantization))
		}
		position, iterator = self.drawWrapLine(target, position, iterator, runeCount, lineBreak)

		// advance line unless on last line without line break
		if lineBreak || !eot {
			lineBreakNth = maxInt(1, lineBreakNth + 1)
			position = self.advanceLine(position, startX, lineBreakNth)
			if position.Y > maxY { break }
		}

		if eot { break }
	}
}

func (self *Renderer) fractDrawWithWrapCenter(target TargetImage, text string, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	panic("unimplemented")
}

// ---- helper functions ----

// Returns nextCount, lineBreak, eot. Versions for centering
// may require actual length and so on. Position must be
// passed already quantized if necessary.
func (self *Renderer) measureWrapLine(iterator strIterator, widthLimit fract.Unit) (fract.Unit, int, bool, bool) {
	var width, x fract.Unit
	var prevGlyphIndex sfnt.GlyphIndex
	var nextCount int
	var lastSafeCount int
	var lineStart bool = true

	// TODO: yeah, better just split all the cases into measureWrapLineLTR and so on
	//       like on regular draws, and have fractDrawWrapLeftLTR, fractDrawWrapLeftRTL, etc.
	horzQuant := fract.Unit(self.state.horzQuantization)
	if self.state.textDirection == LeftToRight {
		for {
			codePoint := iterator.Next()
			if codePoint == -1 { return x, nextCount, false, true }
			nextCount += 1

			if codePoint == '\n' { return x, nextCount, true, false }
			if codePoint == ' ' { lastSafeCount = nextCount }

			// get glyph index
			currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

			// apply kerning unless at line start
			if lineStart {
				lineStart = false
			} else {
				x += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
				x = x.QuantizeUp(horzQuant)
			}

			// advance
			x += self.getOpAdvance(currGlyphIndex)

			// stop if outside wrapLimit
			if x > widthLimit {
				if lastSafeCount == 0 {
					return width, maxInt(nextCount - 1, 1), false, false
				} else {
					return width, lastSafeCount, false, false
				}
			}
			
			// update tracking variables
			width = x
			prevGlyphIndex = currGlyphIndex
		}
	} else { // assume self.state.textDirection == RightToLeft
		for {
			codePoint := iterator.Next()
			if codePoint == -1 { return x, nextCount, false, true }
			nextCount += 1

			if codePoint == '\n' { return x, nextCount, true, false }
			if codePoint == ' ' { lastSafeCount = nextCount }

			// get glyph index
			currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

			// advance
			x += self.getOpAdvance(currGlyphIndex)

			// apply kerning unless at line start
			if lineStart {
				lineStart = false
			} else {
				x += self.getOpKernBetween(currGlyphIndex, prevGlyphIndex)
				x = x.QuantizeUp(horzQuant)
			}

			// stop if outside wrapLimit
			if x > widthLimit {
				if lastSafeCount == 0 {
					return width, maxInt(nextCount - 1, 1), false, false
				} else {
					return width, lastSafeCount, false, false
				}
			}
			
			// update tracking variables
			width = x
			prevGlyphIndex = currGlyphIndex
		}
	}
}

func (self *Renderer) drawWrapLine(target TargetImage, position fract.Point, iterator strIterator, nextCount int, lineBreak bool) (fract.Point, strIterator) {
	var prevGlyphIndex sfnt.GlyphIndex
	prevFractX := position.X.FractShift()
	
	iterCount := nextCount
	if lineBreak { iterCount -= 1 }
	// TODO: again, split and use both for Draw and DrawWithWrap. this is relevant for drawing lines on
	//       centered draw, and I can specialize for ltrStringIterator and rtlStringIterator
	horzQuant := fract.Unit(self.state.horzQuantization)
	if self.state.textDirection == LeftToRight {
		for i := 0; i < nextCount; i++ {
			codePoint := iterator.Next()
			
			// get glyph index
			currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
			
			// apply kerning unless coming from line break
			if i > 0 {
				position.X += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
				position.X  = position.X.QuantizeUp(horzQuant)
			}
	
			newFractX := position.X.FractShift()
			if newFractX != prevFractX {
				self.cacheHandler.NotifyFractChange(position)
				prevFractX = newFractX
			}
	
			// draw glyph
			self.internalGlyphDraw(target, currGlyphIndex, position)
	
			// advance
			position.X += self.getOpAdvance(currGlyphIndex)
	
			// update tracking variables
			prevGlyphIndex = currGlyphIndex
		}
	} else { // assume self.state.textDirection == RightToLeft
		for i := 0; i < nextCount; i++ {
			codePoint := iterator.Next()
			
			// get glyph index
			currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
			
			// advance
			position.X -= self.getOpAdvance(currGlyphIndex)
			position.X  = position.X.QuantizeUp(horzQuant) // quantize
	
			newFractX := position.X.FractShift()
			if newFractX != prevFractX {
				self.cacheHandler.NotifyFractChange(position)
				prevFractX = newFractX
			}

			// draw glyph
			self.internalGlyphDraw(target, currGlyphIndex, position)
	
			// apply kerning unless coming from line break
			if i > 0 {
				position.X -= self.getOpKernBetween(currGlyphIndex, prevGlyphIndex)
				
			}
	
			// update tracking variables
			prevGlyphIndex = currGlyphIndex
		}
	}
	
	// skip explicit line break if necessary
	if lineBreak { _ = iterator.Next() }

	return position, iterator
}
	
