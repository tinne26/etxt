package etxt

import "golang.org/x/image/font/sfnt"

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
		y = (y + ascent - xheight).QuantizeUp(vertQuant)
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
	x = x.QuantizeUp(horzQuant)
	ltr := (self.state.textDirection == LeftToRight)
	switch self.state.align.Horz() {
	case Left:
		if ltr {
			self.fractDrawLeftLTR(target, text, lineBreakNth, x, y, maxBaselineY)
		} else {
			self.fractDrawLeftRTL(target, text, lineBreakNth, x, y, maxBaselineY)
		}
	case Right:
		if ltr {
			self.fractDrawRightLTR(target, text, lineBreakNth, x, y, maxBaselineY)
		} else {
			self.fractDrawRightRTL(target, text, lineBreakNth, x, y, maxBaselineY)
		}
	case HorzCenter:
		reverse := (self.state.textDirection != LeftToRight)
		self.fractDrawCenter(target, text, reverse, lineBreakNth, x, y, maxBaselineY)
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
			position, iv.lineBreakNth = self.advanceLine(position, x, iv.lineBreakNth)
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
			position, iv.lineBreakNth = self.advanceLine(position, x, iv.lineBreakNth)
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
			position, iv.lineBreakNth = self.advanceLine(position, x, iv.lineBreakNth)
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
			position, iv.lineBreakNth = self.advanceLine(position, x, iv.lineBreakNth)
			if position.Y > maxY { break }
		} else {
			position, iv = self.drawGlyphRTL(target, position, codePoint, iv)
		}
	}
}

func (self *Renderer) fractDrawLeft(target TargetImage, text string, reverse bool, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var position fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := self.fractGetQuantization()

	latestFractY := position.Y.FractShift()
	latestFractX := position.X.FractShift().QuantizeUp(horzQuant)
	startX := position.X.QuantizeUp(horzQuant)
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	// neither text direction nor align matter in this context.
	// only font, size and quantization. go traverse the text.
	for {
		codePoint := iterator.Next()
		if codePoint == -1 { break }

		// handle line break case
		if codePoint == '\n' { // line break case
			// move pen position to next line
			if lineBreakNth == -1 { lineBreakNth = 0 }
			lineBreakNth += 1
			position.X = startX
			position.Y += self.getOpLineAdvance(lineBreakNth)
			position.Y = position.Y.QuantizeUp(vertQuant)
			if position.Y > maxY { return } // early return
			newestFractY := position.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				latestFractY = newestFractY
				latestFractX = startX
				if self.cacheHandler != nil {
					self.cacheHandler.NotifyFractChange(position)
				}
			}
			continue
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
		
		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			position.X += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
			position.X = position.X.QuantizeUp(horzQuant) // quantize
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
}

func (self *Renderer) fractDrawRight(target TargetImage, text string, reverse bool, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var position fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := self.fractGetQuantization()

	latestFractY := position.Y.FractShift()
	latestFractX := position.X.FractShift()
	startX := position.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	// neither text direction nor align matter in this context.
	// only font, size and quantization. go traverse the text.
	for {
		codePoint := iterator.Next()
		if codePoint == -1 { break }

		// handle line break case
		if codePoint == '\n' { // line break case
			// move pen position to next line
			if lineBreakNth == -1 { lineBreakNth = 0 }
			lineBreakNth += 1
			position.X = startX
			position.Y += self.getOpLineAdvance(lineBreakNth)
			position.Y = position.Y.QuantizeUp(vertQuant)
			if position.Y > maxY { return } // early return
			newestFractY := position.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				latestFractY = newestFractY
				latestFractX = startX
				if self.cacheHandler != nil {
					self.cacheHandler.NotifyFractChange(position)
				}
			}
			continue
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
		
		// advance
		position.X -= self.getOpAdvance(currGlyphIndex)

		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			position.X -= self.getOpKernBetween(currGlyphIndex, prevGlyphIndex)
		}
		
		// quantize and notify changes
		position.X = position.X.QuantizeUp(horzQuant)
		newestFractX := position.X.FractShift()
		if newestFractX != latestFractX {
			latestFractX = newestFractX
			if self.cacheHandler != nil {
				self.cacheHandler.NotifyFractChange(position)
			}
		}

		// draw glyph
		self.internalGlyphDraw(target, currGlyphIndex, position)
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
	}
}

func (self *Renderer) fractDrawCenter(target TargetImage, text string, reverse bool, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var position fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := self.fractGetQuantization()

	latestFractY := position.Y.FractShift()
	latestFractX := position.X.FractShift()
	centerX := position.X // better not quantize here
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	// traverse text
	initIter  := iterator
	codePoint := iterator.Next()
outerLoop: // TODO: should probably move part of this into a simpler measuring routine
	for {
		if codePoint == '\n' {
			// update fractional position
			newestFractX := position.X.FractShift()
			newestFractY := position.Y.FractShift()
			if newestFractY != latestFractY || newestFractX != latestFractX {
				if self.cacheHandler != nil {
					self.cacheHandler.NotifyFractChange(position)
				}
				latestFractX, latestFractY = newestFractX, newestFractY
			}
			
			// draw previously measured text
			iterator  = initIter
			codePoint = iterator.Next()
			for {
				if codePoint == -1   { break outerLoop }
				if codePoint == '\n' { break }

				// get glyph index
				currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
				
				// apply kerning unless coming from line break
				if lineBreakNth != 0 {
					lineBreakNth = 0
				} else {
					position.X += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
					position.X = position.X.QuantizeUp(horzQuant) // quantize
				}

				newestFractX := position.X.FractShift()
				if newestFractX != latestFractX {
					if self.cacheHandler != nil {
						self.cacheHandler.NotifyFractChange(position)
					}
					latestFractX = newestFractX
				}

				// draw glyph
				self.internalGlyphDraw(target, currGlyphIndex, position)

				// advance
				position.X += self.getOpAdvance(currGlyphIndex)
				
				// update prev glyph and go to next code point
				prevGlyphIndex = currGlyphIndex
				codePoint = iterator.Next()
			}

			// process line breaks
			position.X = centerX
			if lineBreakNth == -1 { lineBreakNth = 0 }
			for codePoint == '\n' {
				// switch to handle line breaks
				lineBreakNth += 1
				position.Y += self.getOpLineAdvance(lineBreakNth)
				position.Y = position.Y.QuantizeUp(vertQuant)
				if position.Y > maxY { return } // early return
				newestFractY = position.Y.FractShift()

				// memorize position and get next rune
				initIter  = iterator
				codePoint = iterator.Next()
			}
			if codePoint == -1 { break outerLoop }
		}

		// compute line width
		var lineWidth fract.Unit
		for {
			if codePoint == -1 || codePoint == '\n' {
				position.X = (centerX - (lineWidth >> 1)).QuantizeUp(horzQuant)
				codePoint = '\n' // fake line break even on end to force last flush
				lineBreakNth = 1
				break
			} else {
				// get glyph index
				currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
				
				// apply kerning unless coming from line break
				if lineBreakNth != 0 {
					lineBreakNth = 0
				} else {
					lineWidth += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
					lineWidth = lineWidth.QuantizeUp(horzQuant) // quantize
				}

				// advance
				lineWidth += self.getOpAdvance(currGlyphIndex)
				
				// update prev glyph and go to next code point
				prevGlyphIndex = currGlyphIndex
				codePoint = iterator.Next()
			}
		}
	}
}
