package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// Same as [Renderer.Draw](), but using a width limit for line wrapping.
// The line wrapping algorithm is a trivial greedy algorithm using
// spaces as the only line breaking points.
//
// The widthLimit must must be given in real units, not logical ones.
// This means that unlike text sizes, the widthLimit won't be internally
// multiplied by the renderer's scale factor.
//
// TODO: mention that centered and last aligns will force a MeasureWithWrap()
// call?
func (self *Renderer) DrawWithWrap(target TargetImage, text string, x, y, widthLimit int) {
	if widthLimit > fract.MaxInt { panic("widthLimit too big, must be <= fract.MaxInt") }
	self.fractDrawWithWrap(target, text, fract.FromInt(x), fract.FromInt(y), fract.FromInt(widthLimit))
}

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
	vertQuant := fract.Unit(self.state.vertQuantization)
	ascent := self.state.fontSizer.Ascent(self.state.activeFont, &self.buffer, self.state.scaledSize)
	switch self.state.align.Vert() {
	case Top:
		y = (y + ascent).QuantizeUp(vertQuant)
	case Midline:
		xheight := self.getSlowOpXHeight()
		y = (y + ascent - xheight).QuantizeUp(vertQuant)
	case VertCenter:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		y = (y + ascent - (height >> 1)).QuantizeUp(vertQuant)
	case Baseline:
		y = y.QuantizeUp(vertQuant)
	case LastBaseline:
		height := self.fractMeasureWithWrap(text, widthLimit).Height()
		lineHeight := self.getOpLineHeight()
		if height <= lineHeight {
			y = (y - height).QuantizeUp(vertQuant)
		} else {
			height -= lineHeight
			y = (y - height).QuantizeUp(vertQuant)
		}
	case Bottom:
		height := self.fractMeasureHeight(text)
		y = (y + ascent - height).QuantizeUp(vertQuant)
	default:
		panic(self.state.align.Vert())
	}

	maxBaselineY := fract.FromInt(bounds.Max.Y) + ascent
	var lineBreakNth int = -1
	// TODO: regular draw has an optimization to skip non-visible portions
	//       of the text, but with max line width it's not so simple, and
	//       the difference it makes should also be smaller. that said, I
	//       should still try to do it when I can, it should still be useful.
	//       Technically, similar optimizations could be added for the
	//       horizontal dimension too, but I think in practice Y is more
	//       important due to scrollable text in UIs, which are more likely
	//       to be poorly optimized. also, with line wrap, the algorithm becomes
	//       really messy if we start adding extra stuff like this on the X
	//       axis...

	// subdelegate to drawWithWrapLTR, drawWithWrapRTL or drawWithWrapCenter
	switch self.state.align.Horz() {
	case Left:
		reverse := (self.state.textDirection != LeftToRight)
		self.fractDrawWithWrapLTR(target, text, reverse, lineBreakNth, x, y, widthLimit, maxBaselineY)
	case Right:
		reverse := (self.state.textDirection != RightToLeft)
		self.fractDrawWithWrapRTL(target, text, reverse, lineBreakNth, x, y, widthLimit, maxBaselineY)
	case HorzCenter:
		reverse := (self.state.textDirection != LeftToRight)
		self.fractDrawWithWrapCenter(target, text, reverse, lineBreakNth, x, y, widthLimit, maxBaselineY)
	default:
		panic(self.state.align.Horz())
	}
}

func (self *Renderer) fractDrawWithWrapLTR(target TargetImage, text string, reverse bool, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
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
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var position fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := self.fractGetQuantization()

	latestFractY := position.Y.FractShift()
	latestFractX := position.X.FractShift().QuantizeUp(horzQuant)
	startX := position.X.QuantizeUp(horzQuant)
	wrapLimit := startX + widthLimit
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	// neither text direction nor align matter in this context.
	// only font, size and quantization. go traverse the text.
	for { // for each word
		nextCount, lineBreak, eot := self.determineWrapLine(iterator, position, wrapLimit)
		
		// draw glyphs for the next line, without checks
		iterCount := nextCount
		if lineBreak { iterCount -= 1 }
		for i := 0; i < nextCount; i++ {
			codePoint := iterator.Next()
			
			// get glyph index
			currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
					
			// apply kerning unless coming from line break
			if lineBreakNth != 0 {
				lineBreakNth = 0
			} else {
				position.X += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
				position.X  = position.X.QuantizeUp(horzQuant) // quantize
			}

			newestFractX := position.X.FractShift()
			if newestFractX != latestFractX {
				self.cacheHandler.NotifyFractChange(position)
				latestFractX = newestFractX
			}

			// draw glyph
			self.internalGlyphDraw(target, currGlyphIndex, position)

			// advance
			position.X += self.getOpAdvance(currGlyphIndex)

			// update tracking variables
			prevGlyphIndex = currGlyphIndex
		}

		// skip explicit line break if necessary
		if lineBreak { _ = iterator.Next() }

		// move pen position to next line if necessary
		if lineBreak || !eot {
			lineBreakNth = maxInt(1, lineBreakNth + 1)
			position.X = startX
			position.Y += self.getOpLineAdvance(lineBreakNth)
			position.Y  = position.Y.QuantizeUp(vertQuant)
			if position.Y > maxY { return } // early return
			newestFractY := position.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				latestFractY = newestFractY
				latestFractX = startX
				if self.cacheHandler != nil {
					self.cacheHandler.NotifyFractChange(position)
				}
			}
		}

		if eot { break }
	}
}

func (self *Renderer) fractDrawWithWrapRTL(target TargetImage, text string, reverse bool, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	panic("unimplemented")
}

func (self *Renderer) fractDrawWithWrapCenter(target TargetImage, text string, reverse bool, lineBreakNth int, x, y, widthLimit, maxY fract.Unit) {
	panic("unimplemented")
}

// ---- helper functions ----

// Returns nextCount, lineBreak, eot. Versions for centering
// may require actual length and so on. Position must be
// passed already quantized if necessary.
func (self *Renderer) determineWrapLine(iterator strIterator, position fract.Point, wrapLimit fract.Unit) (int, bool, bool) {
	var prevGlyphIndex sfnt.GlyphIndex
	var nextCount int
	var lastSafeCount int
	var lineStart bool = true

	for {
		codePoint := iterator.Next()
		if codePoint == -1 { return nextCount, false, true }
		nextCount += 1

		if codePoint == '\n' { return nextCount, true, false }
		if codePoint == ' ' { lastSafeCount = nextCount }

		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)

		// apply kerning unless at line start
		if lineStart {
			lineStart = false
		} else {
			position.X += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
			position.X = position.X.QuantizeUp(fract.Unit(self.state.horzQuantization))
		}

		// advance
		position.X += self.getOpAdvance(currGlyphIndex)

		// stop if outside wrapLimit
		if position.X > wrapLimit {
			if lastSafeCount == 0 {
				return maxInt(nextCount - 1, 1), false, false
			} else {
				return lastSafeCount, false, false
			}
		}

		// update tracking variables
		prevGlyphIndex = currGlyphIndex
	}	
}
