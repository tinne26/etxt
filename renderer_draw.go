package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

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

func (self *Renderer) fractDraw(target TargetImage, text string, x, y fract.Unit) {
	// return directly on superfluous invocations
	if text == "" { return } // return fract.UnitsToPoint(x, y)

	bounds := target.Bounds()
	if bounds.Empty() { return }
	
	// preconditions
	font := self.GetFont()
	if target == nil { panic("can't draw on nil TargetImage") }
	if font == nil { panic("can't draw text with nil font (tip: Renderer.SetFont())") }

	// ensure relevant properties are initialized
	if self.state.fontSizer  == nil { panic("can't draw with a nil sizer (tip: NewRenderer())") }
	if self.state.rasterizer == nil { panic("can't draw with a nil rasterizer (tip: NewRenderer())") }
	sizer := self.state.fontSizer
	vertQuant := fract.Unit(self.state.vertQuantization)

	// adjust Y position
	ascent := sizer.Ascent(font, &self.buffer, self.state.scaledSize)
	switch self.state.align.Vert() {
	case Top:
		y = (y + ascent).QuantizeUp(vertQuant)
	case Midline:
		xheight := self.xheight(font) // note: slower to obtain than the other metrics
		y = (y + ascent - xheight).QuantizeUp(vertQuant)
	case VertCenter:
		height := self.fractMeasureHeight(text)
		y = (y + ascent - (height >> 1)).QuantizeUp(vertQuant)
	case Baseline:
		y = y.QuantizeUp(vertQuant)
	case LastBaseline:
		height := self.fractMeasureHeight(text)
		lineHeight := sizer.LineHeight(font, &self.buffer, self.state.scaledSize)
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

	// skip non-visible portions of the text in the target
	// Note: poorly designed fonts where glyphs can go above their
	//       declared ascent may be clipped. I blame the fonts.
	minBaselineY := fract.FromInt(bounds.Min.Y) - ascent
	maxBaselineY := fract.FromInt(bounds.Max.Y) + ascent
	var lineBreakNth int = -1
	if y < minBaselineY {
		for i, codePoint := range text {
			if codePoint == '\n' {
				if lineBreakNth == -1 { lineBreakNth = 0 }
				lineBreakNth += 1
				y += sizer.LineAdvance(font, &self.buffer, self.state.scaledSize, lineBreakNth)
				y  = y.QuantizeUp(vertQuant)
				if y >= minBaselineY {
					text = text[i + 1 : ]
					break
				}
			}
		}
	}

	// subdelegate to drawLTR, drawRTL or drawCenter
	switch self.state.align.Horz() {
	case Left:
		reverse := (self.state.textDirection != LeftToRight)
		self.fractDrawLTR(target, text, reverse, font, lineBreakNth, x, y, maxBaselineY)
	case Right:
		reverse := (self.state.textDirection != RightToLeft)
		self.fractDrawRTL(target, text, reverse, font, lineBreakNth, x, y, maxBaselineY)
	case HorzCenter:
		reverse := (self.state.textDirection != LeftToRight)
		self.fractDrawCenter(target, text, reverse, font, lineBreakNth, x, y, maxBaselineY)
	default:
		panic(self.state.align.Horz())
	}
}

func (self *Renderer) internalGlyphDraw(target TargetImage, glyphIndex sfnt.GlyphIndex, origin fract.Point, font *sfnt.Font) {
	if self.customDrawFn != nil {
		self.customDrawFn(target, glyphIndex, origin)
	} else {
		mask := self.loadGlyphMask(font, glyphIndex, origin)
		self.defaultDrawFunc(target, origin, mask)
	}
}

func (self *Renderer) fractDrawLTR(target TargetImage, text string, reverse bool, font *sfnt.Font, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var origin fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := self.fractGetQuantization()
	sizer := self.state.fontSizer
	size := self.state.scaledSize

	latestFractY := origin.Y.FractShift()
	latestFractX := origin.X.FractShift().QuantizeUp(horzQuant)
	startX := origin.X.QuantizeUp(horzQuant)
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(origin)
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
			origin.X = startX
			origin.Y += sizer.LineAdvance(font, &self.buffer, size, lineBreakNth)
			origin.Y = origin.Y.QuantizeUp(vertQuant)
			if origin.Y > maxY { return } // early return
			newestFractY := origin.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				latestFractY = newestFractY
				latestFractX = startX
				if self.cacheHandler != nil {
					self.cacheHandler.NotifyFractChange(origin)
				}
			}
			continue
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(font, codePoint)
		
		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			origin.X += sizer.Kern(font, &self.buffer, size, prevGlyphIndex, currGlyphIndex)
			origin.X = origin.X.QuantizeUp(horzQuant) // quantize
		}

		newestFractX := origin.X.FractShift()
		if newestFractX != latestFractX {
			self.cacheHandler.NotifyFractChange(origin)
			latestFractX = newestFractX
		}

		// draw glyph
		self.internalGlyphDraw(target, currGlyphIndex, origin, font)

		// advance
		origin.X += sizer.GlyphAdvance(font, &self.buffer, size, currGlyphIndex)
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
	}
}

func (self *Renderer) fractDrawRTL(target TargetImage, text string, reverse bool, font *sfnt.Font, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var origin fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := self.fractGetQuantization()
	sizer := self.state.fontSizer
	size := self.state.scaledSize

	latestFractY := origin.Y.FractShift()
	latestFractX := origin.X.FractShift()
	startX := origin.X
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(origin)
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
			origin.X = startX
			origin.Y += sizer.LineAdvance(font, &self.buffer, size, lineBreakNth)
			origin.Y = origin.Y.QuantizeUp(vertQuant)
			if origin.Y > maxY { return } // early return
			newestFractY := origin.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				latestFractY = newestFractY
				latestFractX = startX
				if self.cacheHandler != nil {
					self.cacheHandler.NotifyFractChange(origin)
				}
			}
			continue
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(font, codePoint)
		
		// advance
		origin.X -= sizer.GlyphAdvance(font, &self.buffer, size, currGlyphIndex)

		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			origin.X -= sizer.Kern(font, &self.buffer, size, prevGlyphIndex, currGlyphIndex)
		}
		
		// quantize and notify changes
		origin.X = origin.X.QuantizeUp(horzQuant)
		newestFractX := origin.X.FractShift()
		if newestFractX != latestFractX {
			latestFractX = newestFractX
			if self.cacheHandler != nil {
				self.cacheHandler.NotifyFractChange(origin)
			}
		}

		// draw glyph
		self.internalGlyphDraw(target, currGlyphIndex, origin, font)
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
	}

	// TODO: I could store the unquantized pen position so it can be retrieved,
	//       even if it's through a different method. hmmmm...
}

func (self *Renderer) fractDrawCenter(target TargetImage, text string, reverse bool, font *sfnt.Font, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var origin fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := self.fractGetQuantization()
	sizer := self.state.fontSizer
	size := self.state.scaledSize

	latestFractY := origin.Y.FractShift()
	latestFractX := origin.X.FractShift()
	centerX := origin.X // better not quantize here
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(origin)
	}

	// traverse text
	memo := iterator.MemorizePosition()
	codePoint := iterator.Next()
outerLoop:
	for {
		if codePoint == '\n' {
			// update fractional position
			newestFractX := origin.X.FractShift()
			newestFractY := origin.Y.FractShift()
			if newestFractY != latestFractY || newestFractX != latestFractX {
				if self.cacheHandler != nil {
					self.cacheHandler.NotifyFractChange(origin)
				}
				latestFractX, latestFractY = newestFractX, newestFractY
			}
			
			// draw previously measured text
			iterator.RestorePosition(memo)
			codePoint = iterator.Next()
			for {
				if codePoint == -1   { break outerLoop }
				if codePoint == '\n' { break }

				// get glyph index
				currGlyphIndex := self.getGlyphIndex(font, codePoint)
				
				// apply kerning unless coming from line break
				if lineBreakNth != 0 {
					lineBreakNth = 0
				} else {
					origin.X += sizer.Kern(font, &self.buffer, size, prevGlyphIndex, currGlyphIndex)
					origin.X = origin.X.QuantizeUp(horzQuant) // quantize
				}

				newestFractX := origin.X.FractShift()
				if newestFractX != latestFractX {
					if self.cacheHandler != nil {
						self.cacheHandler.NotifyFractChange(origin)
					}
					latestFractX = newestFractX
				}

				// draw glyph
				self.internalGlyphDraw(target, currGlyphIndex, origin, font)

				// advance
				origin.X += sizer.GlyphAdvance(font, &self.buffer, size, currGlyphIndex)
				
				// update prev glyph and go to next code point
				prevGlyphIndex = currGlyphIndex
				codePoint = iterator.Next()
			}

			// process line breaks
			origin.X = centerX
			if lineBreakNth == -1 { lineBreakNth = 0 }
			for codePoint == '\n' {
				// switch to handle line breaks
				lineBreakNth += 1
				origin.Y += sizer.LineAdvance(font, &self.buffer, size, lineBreakNth)
				origin.Y = origin.Y.QuantizeUp(vertQuant)
				if origin.Y > maxY { return } // early return
				newestFractY = origin.Y.FractShift()

				// memorize position and get next rune
				memo = iterator.MemorizePosition()
				codePoint = iterator.Next()
			}
			if codePoint == -1 { break outerLoop }
		}

		// compute line width
		var lineWidth fract.Unit
		for {
			if codePoint == -1 || codePoint == '\n' {
				origin.X = (centerX - (lineWidth >> 1)).QuantizeUp(horzQuant)
				codePoint = '\n' // fake line break even on end to force last flush
				lineBreakNth = 1
				break
			} else {
				// get glyph index
				currGlyphIndex := self.getGlyphIndex(font, codePoint)
				
				// apply kerning unless coming from line break
				if lineBreakNth != 0 {
					lineBreakNth = 0
				} else {
					lineWidth += sizer.Kern(font, &self.buffer, size, prevGlyphIndex, currGlyphIndex)
					lineWidth = lineWidth.QuantizeUp(horzQuant) // quantize
				}

				// advance
				lineWidth += sizer.GlyphAdvance(font, &self.buffer, size, currGlyphIndex)
				
				// update prev glyph and go to next code point
				prevGlyphIndex = currGlyphIndex
				codePoint = iterator.Next()
			}
		}
	}
}
