package etxt

import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt/efixed"

// This file contains the definitions of the text and glyph
// bounding functions for Renderer objects.

// Get the dimensions of the area taken by the given text.
// Intuitively, this matches the shaded area that you see when
// highlighting or selecting text in browsers and text editors.
//
// The results are affected by the renderer's font, size, quantization
// mode, sizer and text direction. If the input text contains \n line
// breaks, then line height and line spacing will also affect the results.
//
// Notice that spilling (content falling outside the returned rect)
// is possible. In general it will be non-existent or very minor, but
// some fancy display or script fonts can really go to crazy places.
// You should also be careful with italics.
func (self *Renderer) SelectionRect(text string) RectSize {
	// (notice that SelectionRect is different from the BoundString() methods
	// offered by golang/x/image/font or ebiten/text, which give you a tight
	// bounding box around the glyph segments composing the text instead
	// through individual glyph control box union, very technical to
	// understand in all its subtleties)

	if text == "" { return RectSize{} }
	width := fixed.Int26_6(0)
	absX  := fixed.Int26_6(0)
	lineBreaksOnly := true
	measureFn :=
		func(currentDot fixed.Point26_6, codePoint rune, _ GlyphIndex) {
			absX = fixedAbs(currentDot.X)
			if absX > width { width = absX }
			if codePoint != '\n' { lineBreaksOnly = false }
		}

	// traverse the string
	vAlign, hAlign := self.tempMeasuringStart()
	dot := self.Traverse(text, fixed.Point26_6{}, measureFn)
	absX = fixedAbs(dot.X)
	if absX > width { width = absX }
	self.tempMeasuringEnd(vAlign, hAlign)

	// obtain height and return
	if self.metrics == nil { self.updateMetrics() }
	height := fixedAbs(dot.Y)
	if !lineBreaksOnly { height += self.metrics.Height }
	return RectSize{ width, height }
}

// Same as [Renderer.SelectionRect](), but taking a slice of glyph indices
// instead of a string.
func (self *Renderer) SelectionRectGlyphs(glyphIndices []GlyphIndex) RectSize {
	if len(glyphIndices) == 0 { return RectSize{} }
	vAlign, hAlign := self.tempMeasuringStart()
	dot := self.TraverseGlyphs(glyphIndices, fixed.Point26_6{}, func(fixed.Point26_6, GlyphIndex) {})
	self.tempMeasuringEnd(vAlign, hAlign)
	if self.metrics == nil { self.updateMetrics() }
	return RectSize{ fixedAbs(dot.X), self.metrics.Height }
}

// During selection rect measuring, using certain aligns like the centered
// ones is extremely inefficient and pointless. So, we disable those
// temporarily and optimize a bit around it. In fact, now Traverse*
// operations depend on centered aligns being used in SelectionRect*
// methods (infinite recursive loops would happen otherwise).
func (self *Renderer) tempMeasuringStart() (VertAlign, HorzAlign) {
	v, h := self.vertAlign, self.horzAlign
	self.vertAlign = Baseline
	self.horzAlign = Left
	if self.direction == RightToLeft {
		self.horzAlign = Right
	}
	return v, h
}

// The counterpart of tempMeasuringStart().
func (self *Renderer) tempMeasuringEnd(origVertAlign VertAlign, origHorzAlign HorzAlign) {
	self.vertAlign = origVertAlign
	self.horzAlign = origHorzAlign
}

// Line breaks are always considered for height, whether they are leading,
// trailing, or even if the input text only contains line breaks.
func (self *Renderer) textHeight(text string) fixed.Int26_6 {
	if text == "" { return 0 }

	// count line breaks
	lineBreakCount := 0
	for _, codePoint := range text {
		if codePoint == '\n' { lineBreakCount += 1 }
	}

	// handle simple no line breaks case
	if self.metrics == nil { self.updateMetrics() }
	if lineBreakCount == 0 { return self.metrics.Height }

	// we have line breaks, get line advance
	lineAdvance := self.GetLineAdvance()
	if lineAdvance < 0 { lineAdvance = -lineAdvance }
	
   lineAdvance = efixed.QuantizeFractUp(lineAdvance, self.vertQuantStep)
	advance := fixed.Int26_6(lineBreakCount)*lineAdvance
	return self.metrics.Height + advance
}

// --- helper methods ---

func fixedAbs(x fixed.Int26_6) fixed.Int26_6 {
	if x >= 0 { return x }
	return -x
}
