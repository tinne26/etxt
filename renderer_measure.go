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
// Notice that overshoot or spilling (content falling outside the returned rect)
// are possible, but in general you shouldn't be worrying about it. Barring
// extreme cases and bad fonts, you should use small margins for your text
// and just trust that typographers know what they are doing with overshoot.
// That being said, italics, fancy display fonts and script fonts are more
// likely to spill and may require bigger margins than other types of fonts.
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
// take into account final spaces in the line.
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
	if self.state.activeFont == nil { panic("can't measure text with font == nil (tip: Renderer.SetFont())") }
	if self.state.fontSizer  == nil { panic("can't measure text with a nil sizer (tip: NewRenderer())") }

	// set up traversal variables
	var position fract.Point
	var maxLineWidth fract.Unit
	var prevGlyphIndex sfnt.GlyphIndex
	var lineBreakNth int = -1
	var hasLineHeight bool
	
	// neither text direction nor align matter in this context.
	// only font, size and quantization. go traverse the text.
	horzQuant, vertQuant := self.fractGetQuantization()
	for _, codePoint := range text {
		// line break case
		if codePoint == '\n' {
			lineBreakNth = maxInt(1, lineBreakNth + 1)

			// update max width
			if position.X > maxLineWidth {
				maxLineWidth = position.X
			}

			// move pen position to next line
			position.X = 0
			position.Y += self.getOpLineAdvance(lineBreakNth)
			position.Y  = position.Y.QuantizeUp(vertQuant)
			continue
		}

		// apply line height if first time hitting a non line break
		if !hasLineHeight {
			position.Y += self.getOpLineHeight()
			position.Y  = position.Y.QuantizeUp(vertQuant)
			hasLineHeight = true
		}

		// regular glyph case
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
		
		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			position.X += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
			position.X  = position.X.QuantizeUp(horzQuant) // quantize
		}

		// advance
		position.X += self.getOpAdvance(currGlyphIndex)
		
		// update previous glyph
		prevGlyphIndex = currGlyphIndex
	}
	
	// compare x for the last line
	if position.X > maxLineWidth {
		maxLineWidth = position.X
	}
	
	// set and quantize final result
	position.X = maxLineWidth.QuantizeUp(horzQuant)

	// return result
	return fract.Rect{ Min: fract.Point{}, Max: position }
}

func (self *Renderer) fractMeasureHeight(text string) fract.Unit {
	// return directly on superfluous invocations
	if text == "" { return 0 }
	
	// preconditions
	if self.state.activeFont == nil { panic("can't measure text height with font == nil (tip: Renderer.SetFont())") }
	if self.state.fontSizer  == nil { panic("can't measure text with a nil sizer (tip: NewRenderer())") }

	// set up traversal variables
	var y fract.Unit
	var lineBreakNth int
	vertQuant := fract.Unit(self.state.vertQuantization)

	for i, codePoint := range text {
		if codePoint == '\n' {
			lineBreakNth += 1
			y += self.getOpLineAdvance(lineBreakNth)
			y  = y.QuantizeUp(vertQuant)
		} else {
			if lineBreakNth == i {
				y += self.getOpLineHeight()
				y  = y.QuantizeUp(vertQuant)
			}
			lineBreakNth = 0
		}
	}

	// return result
	return y
}

func (self *Renderer) fractMeasureWithWrap(text string, widthLimit fract.Unit) fract.Rect {
	// preconditions
	if self.state.activeFont == nil { panic("can't measure text with nil font (tip: Renderer.SetFont())") }
	if self.state.fontSizer  == nil { panic("can't measure text with a nil sizer (tip: NewRenderer())") }
	if widthLimit < 0 { panic("can't use a negative widthLimit") }
	
	// return directly on superfluous invocations
	if text == "" { return fract.Rect{} }
	
	// set up traversal variables
	var position fract.Point // pen position or origin
	var actualMaxWidth fract.Unit
	var prevGlyphIndex sfnt.GlyphIndex
	var lineBreakNth int = -1 // != 0 indicates coming from line break, but -1 is special
	var lastSafeIndex int // for word breaking, a.k.a, after space index
	var lineStartIndex int
	var lastSafeX fract.Unit // within current line
	var hasLineHeight bool
	var index int = 0
	
	// traverse the text
	horzQuant, vertQuant := self.fractGetQuantization()
	for index < len(text) {
		codePoint, runeSize := utf8.DecodeRuneInString(text[index : ])

		// --- line break case ---
		if codePoint == '\n' {
			if position.X > actualMaxWidth { actualMaxWidth = position.X }

			// move pen position to next line
			lineBreakNth = maxInt(1, lineBreakNth + 1)
			position.X = 0
			position.Y += self.getOpLineAdvance(lineBreakNth)
			position.Y  = position.Y.QuantizeUp(vertQuant)

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
			position.Y += self.getOpLineHeight()
			position.Y  = position.Y.QuantizeUp(vertQuant)
			hasLineHeight = true
		}

		// memorize current x as it may be needed later in some cases
		memoX := position.X

		// get glyph index
		currGlyphIndex := self.getGlyphIndex(self.state.activeFont, codePoint)
		
		// apply kerning unless coming from line break
		if lineBreakNth != 0 {
			lineBreakNth = 0
		} else {
			position.X += self.getOpKernBetween(prevGlyphIndex, currGlyphIndex)
			position.X  = position.X.QuantizeUp(horzQuant) // quantize
		}

		// advance
		position.X += self.getOpAdvance(currGlyphIndex)

		// --- operation logic breakdown ---
		// if the glyph fits in the line or is the first in the line, we advance.
		// otherwise, if it's a space, we can discard it and jump to the next line.
		// if it's part of the first word, we use the memorized X and force a jump
		// to the next line too.

		if position.X <= widthLimit { // glyph fits in the line
			if codePoint == ' ' {
				lastSafeIndex = index + 1
				lastSafeX = memoX
			}
		} else { // glyph doesn't fit in the line
			var wrapLineX fract.Unit
			if index == lineStartIndex { // doesn't fit but it's first char, so force it anyway
				wrapLineX = position.X
			} else if codePoint == ' ' { // we can discard space before wrapping
				wrapLineX = memoX
			} else if lastSafeIndex == lineStartIndex { // still on first word, force take up to previous char
				wrapLineX = memoX
			} else { // no fit, roll back
				position.X = lastSafeX
				index = lastSafeIndex
				continue
			}

			// line wrapping case
			if wrapLineX > actualMaxWidth { actualMaxWidth = wrapLineX }
			position.Y += self.getOpLineHeight()
			position.Y  = position.Y.QuantizeUp(vertQuant)
			position.X = 0
			lastSafeIndex  = index + runeSize
			lineStartIndex = lastSafeIndex
		}
		
		// update tracking variables
		prevGlyphIndex = currGlyphIndex
		index += runeSize
	}

	// compare x for the last line
	if position.X > actualMaxWidth {
		actualMaxWidth = position.X
	}

	// quantize result (y is already necessarily quantized)
	position.X = actualMaxWidth.QuantizeUp(horzQuant)

	// return result
	return fract.Rect{ Min: fract.Point{}, Max: position }
}
