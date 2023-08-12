package etxt

import "github.com/tinne26/etxt/fract"

// Returns the dimensions of the area taken by the given text. Intuitively,
// this matches the shaded area that you see when highlighting or selecting
// text in browsers and text editors.
//
// The results are affected by the renderer's font, size, quantization,
// sizer and text direction.
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
// The returned rect dimensions are always quantized, but the width
// doesn't take into account final spaces in the wrapped lines. Notice
// that the returned rect's minimum width may exceed widthLimit if
// the widthLimit is very low and there's some character in the text
// that exceeds it (a single character can't be split into multiple lines).
func (self *Renderer) MeasureWithWrap(text string, widthLimit int) fract.Rect {
	// TODO: the behavior for spaces at the end of the line without any word
	//       afterwards (EOT or line break) is not properly defined. we may
	//       want to improve the code and force it to consider those spaces
	//       as non-wrapping. Could use PeekNext().
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
	// - The returned rect is always quantized.
	
	// preconditions
	if self.state.activeFont == nil { panic("can't measure text with font == nil (tip: Renderer.SetFont())") }
	if self.state.fontSizer  == nil { panic("can't measure text with a nil sizer (tip: NewRenderer())") }

	// main processing
	if text == "" { return fract.Rect{} }
	if self.state.textDirection == LeftToRight {
		return self.fractMeasureLTR(text)
	} else {
		return self.fractMeasureRTL(text)
	}
}

func (self *Renderer) fractMeasureWithWrap(text string, widthLimit fract.Unit) fract.Rect {
	// preconditions
	if self.state.activeFont == nil { panic("can't measure text with nil font (tip: Renderer.SetFont())") }
	if self.state.fontSizer  == nil { panic("can't measure text with a nil sizer (tip: NewRenderer())") }
	if widthLimit < 0 { panic("can't use a negative widthLimit") }
	
	// main processing
	if text == "" { return fract.Rect{} }
	if self.state.textDirection == LeftToRight {
		return self.fractMeasureWrapLTR(text, widthLimit)
	} else {
		return self.fractMeasureWrapRTL(text, widthLimit)
	}
}

// NOTICE: the four functions below are all the same code, with only different
//         helperMeasure* functions. One could argue I should be passing a
//         function directly. Think about it as generics by hand if you want.
//         The helper functions can be found at renderer_measure_helpers.go.

// Preconditions: non-nil font and sizer, non-empty text.
func (self *Renderer) fractMeasureLTR(text string) fract.Rect {
	var iterator ltrStringIterator
	var lastRune rune
	var lineBreakNth int = -1
	var width, height, lineWidth fract.Unit
	var lineBreaksOnly bool = true
	vertQuant := fract.Unit(self.state.vertQuantization)

	for { // measure text line by line
		iterator, lineWidth, _, lastRune = self.helperMeasureLineLTR(iterator, text)
		if lineWidth > 0 {
			if lineWidth > width { width = lineWidth }
			lineBreaksOnly = false
			lineBreakNth = 0
		}
		if lastRune == -1 { break }
		lineBreakNth = maxInt(1, lineBreakNth + 1)
		height = (height + self.getOpLineAdvance(lineBreakNth)).QuantizeUp(vertQuant)
	}
	
	if !lineBreaksOnly { height = (height + self.getOpLineHeight()).QuantizeUp(vertQuant) }
	width = width.QuantizeUp(fract.Unit(self.state.horzQuantization))
	return fract.Rect{ Max: fract.UnitsToPoint(width, height) }
}

// Preconditions: non-nil font and sizer, non-empty text.
func (self *Renderer) fractMeasureRTL(text string) fract.Rect {
	var iterator ltrStringIterator
	var lastRune rune
	var lineBreakNth int = -1
	var width, height, lineWidth fract.Unit
	var lineBreaksOnly bool = true
	vertQuant := fract.Unit(self.state.vertQuantization)

	for { // measure text line by line
		iterator, lineWidth, _, lastRune = self.helperMeasureLineReverseLTR(iterator, text)
		if lineWidth > 0 {
			if lineWidth > width { width = lineWidth }
			lineBreaksOnly = false
			lineBreakNth = 0
		}
		if lastRune == -1 { break }
		lineBreakNth = maxInt(1, lineBreakNth + 1)
		height = (height + self.getOpLineAdvance(lineBreakNth)).QuantizeUp(vertQuant)
	}
	
	if !lineBreaksOnly { height = (height + self.getOpLineHeight()).QuantizeUp(vertQuant) }
	width = width.QuantizeUp(fract.Unit(self.state.horzQuantization))
	return fract.Rect{ Max: fract.UnitsToPoint(width, height) }
}

// Preconditions: non-nil font and sizer, non-empty text.
func (self *Renderer) fractMeasureWrapLTR(text string, widthLimit fract.Unit) fract.Rect {
	var iterator ltrStringIterator
	var lastRune rune
	var lineBreakNth int = -1
	var width, height, lineWidth fract.Unit
	var lineBreaksOnly bool = true
	vertQuant := fract.Unit(self.state.vertQuantization)

	for { // measure text line by line
		iterator, lineWidth, _, lastRune = self.helperMeasureWrapLineLTR(iterator, text, widthLimit)
		if lineWidth > 0 {
			if lineWidth > width { width = lineWidth }
			lineBreaksOnly = false
			lineBreakNth = 0
		}
		if lastRune == -1 { break }
		lineBreakNth = maxInt(1, lineBreakNth + 1)
		height = (height + self.getOpLineAdvance(lineBreakNth)).QuantizeUp(vertQuant)
	}
	
	if !lineBreaksOnly { height = (height + self.getOpLineHeight()).QuantizeUp(vertQuant) }
	width = width.QuantizeUp(fract.Unit(self.state.horzQuantization))
	return fract.Rect{ Max: fract.UnitsToPoint(width, height) }
}

// failing with qt = 64, align = (Baseline | Left), dir = RightToLeft
// Preconditions: non-nil font and sizer, non-empty text.
func (self *Renderer) fractMeasureWrapRTL(text string, widthLimit fract.Unit) fract.Rect {
	var iterator ltrStringIterator
	var lastRune rune
	var lineBreakNth int = -1
	var width, height, lineWidth fract.Unit
	var lineBreaksOnly bool = true
	vertQuant := fract.Unit(self.state.vertQuantization)

	for { // measure text line by line
		iterator, lineWidth, _, lastRune = self.helperMeasureWrapLineReverseLTR(iterator, text, widthLimit)
		if lineWidth > 0 {
			if lineWidth > width { width = lineWidth }
			lineBreaksOnly = false
			lineBreakNth = 0
		}
		if lastRune == -1 { break }
		lineBreakNth = maxInt(1, lineBreakNth + 1)
		height = (height + self.getOpLineAdvance(lineBreakNth)).QuantizeUp(vertQuant)
	}
	
	if !lineBreaksOnly { height = (height + self.getOpLineHeight()).QuantizeUp(vertQuant) }
	width = width.QuantizeUp(fract.Unit(self.state.horzQuantization))
	return fract.Rect{ Max: fract.UnitsToPoint(width, height) }
}
