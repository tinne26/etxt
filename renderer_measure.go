package etxt

import "unicode/utf8"

import "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/fract"

// Returns the dimensions of the area taken by the given text. Intuitively,
// this matches the shaded area that you see when highlighting or selecting
// text in browsers and text editors.
//
// The results are affected by the renderer's font, size, quantization and
// sizer.
// 
// Notice that spilling (content falling outside the returned rect) is possible.
// In general it will be non-existent or very minor, but italics, fancy display
// fonts and script fonts are common offenders that you may want to watch out
// for.
func (self *Renderer) Measure(text string) fract.Rect {
	return self.fractMeasure(text)
}

// Same as [Renderer.Measure](), but using a width limit for line wrapping.
// Typically used in conjunction with [Renderer.DrawWithWrap]().
//
// The widthLimit must must be given in real units, not logical ones.
// This means that unlike text sizes, the widthLimit won't be internally
// multiplied by the renderer's scale factor.
//
// The returned rect dimensions are always quantized, but the width doesn't
// take into account final spaces in the line (TODO: spaces thingie unimplemented).
func (self *Renderer) MeasureWithWrap(text string, widthLimit int) fract.Rect {
	if widthLimit > fract.MaxInt { panic("widthLimit too big, must be <= fract.MaxInt") }
	return self.fractMeasureWithWrap(text, fract.FromInt(widthLimit))
}

// ---- underlying implementations ----

func (self *Renderer) fractMeasure(text string) fract.Rect {
	// Notes on quirkiness:
	// - Consecutive line breaks are vertically quantized
	//   not because that's more correct in isolation, but
	//   because it's more consistent if multiple paragraphs
	//   are placed side by side with different line breaks.
	// - The rect is always measured quantized.

	// return directly on superfluous invocations
	if text == "" { return fract.Rect{} }
	
	// preconditions
	font := self.GetFont()
	if font == nil { panic("can't measure text with font == nil (tip: Renderer.SetFont())") }
	
	// ensure relevant properties are initialized
	if self.missingBasicProps() { self.initBasicProps() }
	self.initSizer()

	// set up traversal variables
	var origin fract.Point
	var maxLineWidth fract.Unit
	var prevGlyphIndex sfnt.GlyphIndex
	var lineBreakNth int = -1
	var hasLineHeight bool
	horzQuant, vertQuant := fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)

	// neither text direction nor align matter in this context.
	// only font, size and quantization. go traverse the text.
	for _, codePoint := range text {
		// line break case
		if codePoint == '\n' {
			if lineBreakNth == -1 { lineBreakNth = 0 }
			lineBreakNth += 1

			// update max width
			if origin.X > maxLineWidth {
				maxLineWidth = origin.X
			}

			// move pen position to next line
			origin.X = 0
			origin.Y += self.fontSizer.LineAdvance(font, &self.buffer, self.scaledSize, lineBreakNth)
			origin.Y = origin.Y.QuantizeUp(vertQuant)
			continue
		}

		// apply line height if first time hitting a non line break
		if !hasLineHeight {
			origin.Y += self.fontSizer.LineHeight(font, &self.buffer, self.scaledSize)
			origin.Y  = origin.Y.QuantizeUp(vertQuant)
			hasLineHeight = true
		}

		// regular glyph case
		currGlyphIndex := self.getGlyphIndex(font, codePoint)
		
		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			origin.X += self.fontSizer.Kern(font, &self.buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
			origin.X = origin.X.QuantizeUp(horzQuant) // quantize
		}

		// advance
		origin.X += self.fontSizer.GlyphAdvance(font, &self.buffer, self.scaledSize, currGlyphIndex)
		
		// update previous glyph
		prevGlyphIndex = currGlyphIndex
	}
	
	// compare x for the last line
	if origin.X > maxLineWidth {
		maxLineWidth = origin.X
	}
	
	// set and quantize final result
	origin.X = maxLineWidth.QuantizeUp(horzQuant)

	// return result
	return fract.Rect{ Min: fract.Point{}, Max: origin }
}

func (self *Renderer) fractMeasureHeight(text string) fract.Unit {
	// return directly on superfluous invocations
	if text == "" { return 0 }
	
	// preconditions
	font := self.GetFont()
	if font == nil { panic("can't measure text height with font == nil (tip: Renderer.SetFont())") }
	
	// ensure relevant properties are initialized
	if self.missingBasicProps() { self.initBasicProps() }
	self.initSizer()

	// set up traversal variables
	var y fract.Unit
	var lineBreakNth int
	vertQuant := fract.Unit(self.vertQuantization)

	for i, codePoint := range text {
		if codePoint == '\n' {
			lineBreakNth += 1
			y += self.fontSizer.LineAdvance(font, &self.buffer, self.scaledSize, lineBreakNth)
			y  = y.QuantizeUp(vertQuant)
		} else {
			if lineBreakNth == i {
				y += self.fontSizer.LineHeight(font, &self.buffer, self.scaledSize)
				y  = y.QuantizeUp(vertQuant)
			}
			lineBreakNth = 0
		}
	}

	// return result
	return y
}

func (self *Renderer) fractMeasureWithWrap(text string, widthLimit fract.Unit) fract.Rect {
	// Notes on quirkiness:
	// - It's unclear whether spaces ' ' at end of line due to 
	//   wrapping should be considered for the rect width or not
	//   when they are kept. I guess they shouldn't, in case we
	//   want to optimize the widthLimit in some extreme way...
	//   Yeah, I guess I should change this in the future. TODO.

	// preconditions
	font := self.GetFont()
	if font == nil { panic("can't measure text with nil font") }
	if widthLimit < 0 { panic("can't use a negative widthLimit") }
	
	// return directly on superfluous invocations
	if text == "" { return fract.Rect{} }
	
	// ensure relevant properties are initialized
	if self.missingBasicProps() { self.initBasicProps() }
	self.initSizer()
	
	// set up traversal variables
	var origin fract.Point
	var maxLineWidth fract.Unit
	var prevGlyphIndex sfnt.GlyphIndex
	var lineBreakNth int = -1
	var lastSafeIndex int // for word breaking, a.k.a, after space index
	var lineStartIndex int
	var lastSafeX fract.Unit // within current line
	var hasLineHeight bool
	horzQuant, vertQuant := fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)
	var index int = 0

	// traverse the text
	for index < len(text) {
		codePoint, runeSize := utf8.DecodeRuneInString(text[index : ])

		// --- line break case ---
		if codePoint == '\n' {
			if origin.X > maxLineWidth { maxLineWidth = origin.X }

			// move pen position to next line
			if lineBreakNth == -1 { lineBreakNth = 0 }
			lineBreakNth += 1
			origin.X = 0
			origin.Y += self.fontSizer.LineAdvance(font, &self.buffer, self.scaledSize, lineBreakNth)
			origin.Y = origin.Y.QuantizeUp(vertQuant)

			// keep going
			index += runeSize
			lastSafeX = 0
			lastSafeIndex = index
			lineStartIndex = index
			continue
		}

		// --- regular character case ---

		// apply line height if first time hitting a non line break
		if !hasLineHeight {
			origin.Y += self.fontSizer.LineHeight(font, &self.buffer, self.scaledSize)
			origin.Y  = origin.Y.QuantizeUp(vertQuant)
			hasLineHeight = true
		}

		// memorize this as it may be needed later in some cases
		memoX := origin.X

		// get glyph index
		currGlyphIndex := self.getGlyphIndex(font, codePoint)
		
		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			origin.X += self.fontSizer.Kern(font, &self.buffer, self.scaledSize, prevGlyphIndex, currGlyphIndex)
			origin.X = origin.X.QuantizeUp(horzQuant) // quantize
		}

		// advance
		origin.X += self.fontSizer.GlyphAdvance(font, &self.buffer, self.scaledSize, currGlyphIndex)

		// --- operation logic breakdown ---
		// if the glyph fits in the line or is the first in the line, we advance.
		// otherwise, if it's a space, we can discard it and jump to the next line.
		// if it's part of the first word, we use the memorized X and force a jump
		// to the next line too.

		if origin.X <= widthLimit { // glyph fits in the line
			if codePoint == ' ' {
				lastSafeIndex = index + 1
				lastSafeX = memoX
			}
		} else { // glyph doesn't fit in the line
			var wrapLinePoint fract.Unit
			if index == lineStartIndex { // doesn't fit but it's first char, so force it anyway
				wrapLinePoint = origin.X
			} else if codePoint == ' ' { // we can discard space before wrapping
				wrapLinePoint = memoX
			} else if lastSafeIndex == lineStartIndex { // still on first word, force take up to previous char
				wrapLinePoint = memoX
			} else { // no fit, roll back
				origin.X = lastSafeX
				index = lastSafeIndex
				continue
			}

			// line wrapping case
			if wrapLinePoint > maxLineWidth { maxLineWidth = wrapLinePoint }
			origin.Y += self.fontSizer.LineHeight(font, &self.buffer, self.scaledSize)
			origin.Y = origin.Y.QuantizeUp(vertQuant)
			origin.X = 0
			lastSafeIndex  = index + runeSize
			lineStartIndex = index + runeSize
		} 
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
		index += runeSize
	}

	// compare x for the last line
	if origin.X > maxLineWidth {
		maxLineWidth = origin.X
	}

	// quantize result
	origin.X = maxLineWidth.QuantizeUp(horzQuant)

	// return result
	return fract.Rect{ Min: fract.Point{}, Max: origin }
}
