package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// Draws the given text with the current configuration (font, size, color,
// target, etc). The position at which the text will be drawn depends on
// the given pixel coordinates and the renderer's align (see
// [Renderer.SetAlign]() rules).
//
// The returned value is the last unquantized dot position, only
// relevant for advanced use-cases.
//
// Missing glyphs in the current font will cause the renderer to panic.
// Consider using [font.GetMissingRunes]() if you need to make your
// system more robust.
//
// // [font.GetMissingRunes]: https://pkg.go.dev/github.com/tinne26/etxt/font#GetMissingRunes
func (self *Renderer) Draw(target TargetImage, text string, x, y int) fract.Point {
	return self.fractDraw(target, text, fract.FromInt(x), fract.FromInt(y))
}

func (self *Renderer) fractDraw(target TargetImage, text string, x, y fract.Unit) fract.Point {
	// return directly on superfluous invocations
	if text == "" { return fract.UnitsToPoint(x, y) }
	
	// preconditions
	font := self.GetFont()
	if target == nil { panic("can't draw on nil TargetImage") }
	if font == nil { panic("can't draw text with font == nil (tip: Renderer.SetFont())") }

	// ensure relevant properties are initialized
	if self.missingBasicProps() { self.initBasicProps() }
	self.initSizer()
	self.initRasterizer()

	// adjust Y position
	switch self.align.Vert() {
	case Top:
		y += self.fontSizer.Ascent(font, &self.Buffer, self.scaledSize)
		y = y.QuantizeUp(fract.Unit(self.vertQuantization))
	case YCenter:
		height := self.fractMeasureHeight(text)
		y -= (height >> 1)
		y += self.fontSizer.Ascent(font, &self.Buffer, self.scaledSize)
		y = y.QuantizeUp(fract.Unit(self.vertQuantization))
	case Baseline:
		y = y.QuantizeUp(fract.Unit(self.vertQuantization))
	case Bottom:
		height := self.fractMeasureHeight(text)
		y -= height
		y += self.fontSizer.Ascent(font, &self.Buffer, self.scaledSize)
		y = y.QuantizeUp(fract.Unit(self.vertQuantization))
	default:
		panic(self.align.Vert())
	}

	// subdelegate to drawLTR, drawRTL or drawCenter
	dir := self.complexGetDirection()
	switch self.align.Horz() {
	case Left:
		reverse := (dir != LeftToRight)
		return self.fractDrawLTR(target, text, reverse, font, x, y)
	case Right:
		reverse := (dir != RightToLeft)
		return self.fractDrawRTL(target, text, reverse, font, x, y)
	case XCenter:
		reverse := (dir != LeftToRight)
		return self.fractDrawCenter(target, text, reverse, font, x, y)
	default:
		panic(self.align.Horz())
	}
}

func (self *Renderer) fractDrawLTR(target TargetImage, text string, reverse bool, font *sfnt.Font, x, y fract.Unit) fract.Point {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var dot fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	var lineBreakNth int = -1
	horzQuant, vertQuant := fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)

	latestFractY := dot.Y.FractShift()
	latestFractX := dot.X.FractShift().QuantizeUp(horzQuant)
	startX := dot.X.QuantizeUp(horzQuant)
	self.cacheHandler.NotifyFractChange(dot)

	// neither text direction nor align matter in this context.
	// only font, size and quantization. go traverse the text.
	for {
		codePoint := iterator.Next()
		if codePoint == -1 { break }

		// handle line break case
		if codePoint == '\n' { // line break case
			// move dot to next line
			if lineBreakNth == -1 { lineBreakNth = 0 }
			lineBreakNth += 1
			dot.X = startX
			dot.Y += self.fontSizer.LineAdvance(font, &self.Buffer, self.scaledSize, lineBreakNth)
			dot.Y = dot.Y.QuantizeUp(vertQuant)
			newestFractY := dot.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				self.cacheHandler.NotifyFractChange(dot)
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
			dot.X += self.fontSizer.Kern(font, &self.Buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
			dot.X = dot.X.QuantizeUp(horzQuant) // quantize
		}

		newestFractX := dot.X.FractShift()
		if newestFractX != latestFractX {
			self.cacheHandler.NotifyFractChange(dot)
		}
		latestFractX = newestFractX

		// draw glyph
		mask := self.loadGlyphMask(font, currGlyphIndex, dot)
		self.defaultDrawFunc(target, dot, mask)

		// advance
		dot.X += self.fontSizer.GlyphAdvance(font, &self.Buffer, self.scaledSize, currGlyphIndex)
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
	}

	// return unquantized dot
	return dot
}

func (self *Renderer) fractDrawRTL(target TargetImage, text string, reverse bool, font *sfnt.Font, x, y fract.Unit) fract.Point {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var dot fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	var lineBreakNth int = -1
	horzQuant, vertQuant := fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)

	latestFractY := dot.Y.FractShift()
	latestFractX := dot.X.FractShift()
	startX := dot.X
	self.cacheHandler.NotifyFractChange(dot)

	// neither text direction nor align matter in this context.
	// only font, size and quantization. go traverse the text.
	for {
		codePoint := iterator.Next()
		if codePoint == -1 { break }

		// handle line break case
		if codePoint == '\n' { // line break case
			// move dot to next line
			if lineBreakNth == -1 { lineBreakNth = 0 }
			lineBreakNth += 1
			dot.X = startX
			dot.Y += self.fontSizer.LineAdvance(font, &self.Buffer, self.scaledSize, lineBreakNth)
			dot.Y = dot.Y.QuantizeUp(vertQuant)
			newestFractY := dot.Y.FractShift()
			if newestFractY != latestFractY || startX != latestFractX {
				latestFractY = newestFractY
				latestFractX = startX
				self.cacheHandler.NotifyFractChange(dot)
			}			
			continue
		}
		
		// get glyph index
		currGlyphIndex := self.getGlyphIndex(font, codePoint)
		
		// advance
		dot.X -= self.fontSizer.GlyphAdvance(font, &self.Buffer, self.scaledSize, currGlyphIndex)

		newestFractX := dot.X.FractShift()
		if newestFractX != latestFractX {
			latestFractX = newestFractX
			self.cacheHandler.NotifyFractChange(dot)
		}

		// draw glyph
		mask := self.loadGlyphMask(font, currGlyphIndex, dot)
		self.defaultDrawFunc(target, dot, mask)

		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			dot.X -= self.fontSizer.Kern(font, &self.Buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
			dot.X = dot.X.QuantizeUp(horzQuant) // quantize
		}
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
	}

	// return unquantized dot
	return dot
}

func (self *Renderer) fractDrawCenter(target TargetImage, text string, reverse bool, font *sfnt.Font, x, y fract.Unit) fract.Point {
	// create string iterator
	iterator := newStrIterator(text, reverse)

	// set up traversal variables
	var dot fract.Point = fract.UnitsToPoint(x, y)
	var prevGlyphIndex sfnt.GlyphIndex
	var lineBreakNth int = -1
	horzQuant, vertQuant := fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)

	latestFractY := dot.Y.FractShift()
	latestFractX := dot.X.FractShift()
	centerX := dot.X // better not quantize here
	self.cacheHandler.NotifyFractChange(dot)

	// traverse text
	memo := iterator.MemorizePosition()
	codePoint := iterator.Next()
outerLoop:
	for {
		if codePoint == '\n' {
			// update fractional position
			newestFractX := dot.X.FractShift()
			newestFractY := dot.Y.FractShift()
			if newestFractY != latestFractY || newestFractX != latestFractX {
				self.cacheHandler.NotifyFractChange(dot)
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
				if lineBreakNth > 0 {
					lineBreakNth = 0
				} else {
					dot.X += self.fontSizer.Kern(font, &self.Buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
					dot.X = dot.X.QuantizeUp(horzQuant) // quantize
				}

				newestFractX := dot.X.FractShift()
				if newestFractX != latestFractX {
					self.cacheHandler.NotifyFractChange(dot)
					latestFractX = newestFractX
				}

				// draw glyph
				mask := self.loadGlyphMask(font, currGlyphIndex, dot)
				self.defaultDrawFunc(target, dot, mask)

				// advance
				dot.X += self.fontSizer.GlyphAdvance(font, &self.Buffer, self.scaledSize, currGlyphIndex)
				
				// update prev glyph and go to next code point
				prevGlyphIndex = currGlyphIndex
				codePoint = iterator.Next()
			}

			// process line breaks
			dot.X = centerX
			if lineBreakNth == -1 { lineBreakNth = 0 }
			for codePoint == '\n' {
				// switch to handle line breaks
				lineBreakNth += 1
				dot.Y += self.fontSizer.LineAdvance(font, &self.Buffer, self.scaledSize, lineBreakNth)
				dot.Y = dot.Y.QuantizeUp(vertQuant)
				newestFractY = dot.Y.FractShift()

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
				dot.X = (centerX - (lineWidth >> 1)).QuantizeUp(horzQuant)
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
					lineWidth += self.fontSizer.Kern(font, &self.Buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
					lineWidth = lineWidth.QuantizeUp(horzQuant) // quantize
				}

				// advance
				lineWidth += self.fontSizer.GlyphAdvance(font, &self.Buffer, self.scaledSize, currGlyphIndex)
				
				// update prev glyph and go to next code point
				prevGlyphIndex = currGlyphIndex
				codePoint = iterator.Next()
			}
		}
	}

	// return unquantized dot
	return dot
}
