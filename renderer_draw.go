package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// Draws the given text with the current configuration (font, size, color,
// target, etc). The position at which the text will be drawn depends on
// the given pixel coordinates and the renderer's align (see
// [Renderer.SetAlign]() rules).
//
// Missing glyphs in the current font will cause the renderer to panic.
// Consider using [font.GetMissingRunes]() if you need to make your
// system more robust.
//
// [font.GetMissingRunes]: https://pkg.go.dev/github.com/tinne26/etxt/font#GetMissingRunes
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
	if font == nil { panic("can't draw text with font == nil (tip: Renderer.SetFont())") }

	// ensure relevant properties are initialized
	if self.missingBasicProps() { self.initBasicProps() }
	self.initSizer()
	self.initRasterizer()

	// adjust Y position
	ascent := self.fontSizer.Ascent(font, &self.buffer, self.scaledSize)
	switch self.align.Vert() {
	case Top:
		y = (y + ascent).QuantizeUp(fract.Unit(self.vertQuantization))
	case TopBaseline:
		y = y.QuantizeUp(fract.Unit(self.vertQuantization))
	case YCenter:
		height := self.fractMeasureHeight(text)
		y = (y + ascent - (height >> 1)).QuantizeUp(fract.Unit(self.vertQuantization))
	case BottomBaseline:
		height := self.fractMeasureHeight(text)
		lineHeight := self.fontSizer.LineHeight(font, &self.buffer, self.scaledSize)
		if height <= lineHeight {
			y = (y - height).QuantizeUp(fract.Unit(self.vertQuantization))
		} else {
			height -= lineHeight
			y = (y - height).QuantizeUp(fract.Unit(self.vertQuantization))
		}
	case Bottom:
		height := self.fractMeasureHeight(text)
		y = (y + ascent - height).QuantizeUp(fract.Unit(self.vertQuantization))
	default:
		panic(self.align.Vert())
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
				y += self.fontSizer.LineAdvance(font, &self.buffer, self.scaledSize, lineBreakNth)
				y  = y.QuantizeUp(fract.Unit(self.vertQuantization))
				if y >= minBaselineY {
					text = text[i + 1 : ]
					break
				}
			}
		}
	}

	// subdelegate to drawLTR, drawRTL or drawCenter
	dir := self.complexGetDirection()
	switch self.align.Horz() {
	case Left:
		reverse := (dir != LeftToRight)
		self.fractDrawLTR(target, text, reverse, font, lineBreakNth, x, y, maxBaselineY)
	case Right:
		reverse := (dir != RightToLeft)
		self.fractDrawRTL(target, text, reverse, font, lineBreakNth, x, y, maxBaselineY)
	case XCenter:
		reverse := (dir != LeftToRight)
		self.fractDrawCenter(target, text, reverse, font, lineBreakNth, x, y, maxBaselineY)
	default:
		panic(self.align.Horz())
	}
}

func (self *Renderer) fractDrawLTR(target TargetImage, text string, reverse bool, font *sfnt.Font, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var origin fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)

	latestFractY := origin.Y.FractShift()
	latestFractX := origin.X.FractShift().QuantizeUp(horzQuant)
	startX := origin.X.QuantizeUp(horzQuant)
	self.cacheHandler.NotifyFractChange(origin)

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
			origin.Y += self.fontSizer.LineAdvance(font, &self.buffer, self.scaledSize, lineBreakNth)
			origin.Y = origin.Y.QuantizeUp(vertQuant)
			if origin.Y > maxY { return } // early return
			newestFractY := origin.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				self.cacheHandler.NotifyFractChange(origin)
			}
			latestFractY = newestFractY
			latestFractX = startX
			continue
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(font, codePoint)
		
		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			origin.X += self.fontSizer.Kern(font, &self.buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
			origin.X = origin.X.QuantizeUp(horzQuant) // quantize
		}

		newestFractX := origin.X.FractShift()
		if newestFractX != latestFractX {
			self.cacheHandler.NotifyFractChange(origin)
		}
		latestFractX = newestFractX

		// draw glyph
		if self.customDrawFn != nil {
			self.customDrawFn(target, currGlyphIndex, origin)
		} else {
			mask := self.loadGlyphMask(font, currGlyphIndex, origin)
			self.defaultDrawFunc(target, origin, mask)
		}

		// advance
		origin.X += self.fontSizer.GlyphAdvance(font, &self.buffer, self.scaledSize, currGlyphIndex)
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
	}

	// return unquantized pen position
	// return origin
}

func (self *Renderer) fractDrawRTL(target TargetImage, text string, reverse bool, font *sfnt.Font, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var origin fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)

	latestFractY := origin.Y.FractShift()
	latestFractX := origin.X.FractShift()
	startX := origin.X
	self.cacheHandler.NotifyFractChange(origin)

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
			origin.Y += self.fontSizer.LineAdvance(font, &self.buffer, self.scaledSize, lineBreakNth)
			origin.Y = origin.Y.QuantizeUp(vertQuant)
			if origin.Y > maxY { return } // early return
			newestFractY := origin.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				latestFractY = newestFractY
				latestFractX = startX
				self.cacheHandler.NotifyFractChange(origin)
			}			
			continue
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(font, codePoint)
		
		// advance
		origin.X -= self.fontSizer.GlyphAdvance(font, &self.buffer, self.scaledSize, currGlyphIndex)

		newestFractX := origin.X.FractShift()
		if newestFractX != latestFractX {
			latestFractX = newestFractX
			self.cacheHandler.NotifyFractChange(origin)
		}

		// draw glyph
		if self.customDrawFn != nil {
			self.customDrawFn(target, currGlyphIndex, origin)
		} else {
			mask := self.loadGlyphMask(font, currGlyphIndex, origin)
			self.defaultDrawFunc(target, origin, mask)
		}

		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			origin.X -= self.fontSizer.Kern(font, &self.buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
			origin.X = origin.X.QuantizeUp(horzQuant) // quantize
		}
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
	}

	// return unquantized pen position
	// return origin
}

func (self *Renderer) fractDrawCenter(target TargetImage, text string, reverse bool, font *sfnt.Font, lineBreakNth int, x, y, maxY fract.Unit) {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var origin fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	horzQuant, vertQuant := fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)

	latestFractY := origin.Y.FractShift()
	latestFractX := origin.X.FractShift()
	centerX := origin.X // better not quantize here
	self.cacheHandler.NotifyFractChange(origin)

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
				self.cacheHandler.NotifyFractChange(origin)
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
					origin.X += self.fontSizer.Kern(font, &self.buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
					origin.X = origin.X.QuantizeUp(horzQuant) // quantize
				}

				newestFractX := origin.X.FractShift()
				if newestFractX != latestFractX {
					self.cacheHandler.NotifyFractChange(origin)
					latestFractX = newestFractX
				}

				// draw glyph
				if self.customDrawFn != nil {
					self.customDrawFn(target, currGlyphIndex, origin)
				} else {
					mask := self.loadGlyphMask(font, currGlyphIndex, origin)
					self.defaultDrawFunc(target, origin, mask)
				}

				// advance
				origin.X += self.fontSizer.GlyphAdvance(font, &self.buffer, self.scaledSize, currGlyphIndex)
				
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
				origin.Y += self.fontSizer.LineAdvance(font, &self.buffer, self.scaledSize, lineBreakNth)
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
					lineWidth += self.fontSizer.Kern(font, &self.buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
					lineWidth = lineWidth.QuantizeUp(horzQuant) // quantize
				}

				// advance
				lineWidth += self.fontSizer.GlyphAdvance(font, &self.buffer, self.scaledSize, currGlyphIndex)
				
				// update prev glyph and go to next code point
				prevGlyphIndex = currGlyphIndex
				codePoint = iterator.Next()
			}
		}
	}

	// return unquantized pen position
	// return origin
}
