package etxt

import "golang.org/x/image/math/fixed"

// TODO: add tests comparing with Draw().

// Feeds are the lowest level mechanism to draw text in etxt,
// allowing the user to issue each glyph draw call individually
// and modifying positions or configurations in between.
//
// As a rule of thumb, you should only resort to feeds if
// neither renderer's Draw* nor Traverse* methods give you
// enough control to do what you want. Make sure you are
// well acquainted with those methods first.
//
// Valid Feeds can only be created through [Renderer.NewFeed]().
type Feed struct {
	renderer *Renderer        // associated renderer
	Position fixed.Point26_6  // the feed's working position
	PrevGlyphIndex GlyphIndex // previous glyph index. used for kern.
	HasPrevGlyph bool         // false after line breaks and others. used for kern.
	LineBreakX fixed.Int26_6  // the x coordinate set after a line break
}

// Draws the given rune and advances the Feed's position.
//
// The drawing configuration is taken from the Feed's associated renderer.
//
// Quantization will be checked before every Draw operation and adjusted
// if necessary (even vertical quantization).
func (self *Feed) Draw(codePoint rune) {
	self.DrawGlyph(self.renderer.getGlyphIndex(codePoint))
}

// Same as Draw, but taking a glyph index instead of a rune.
func (self *Feed) DrawGlyph(glyphIndex GlyphIndex) {
	self.traverseGlyph(glyphIndex,
		func(dot fixed.Point26_6) {
				mask := self.renderer.LoadGlyphMask(glyphIndex, dot)
				self.renderer.DefaultDrawFunc(dot, mask, glyphIndex)
		})
}

// Draws the given shape and advances the Feed's position based on
// the image bounds width (rect.Dx). Notice that shapes are not cached,
// so they may be expensive to use. See also DrawImage.
// func (self *Feed) DrawShape(shape emask.Shape) {
//
// }

// TODO: ebiten version vs gtxt version
// func (self *Feed) DrawImage(img image.Image)

// Advances the Feed's position without drawing anything.
func (self *Feed) Advance(codePoint rune) {
	// Note: while this method may seem superfluous, at least it can be
	//       useful when drawing paragraphs at an abitrary scroll position.
	//       Efficient implementations would split by lines from the start,
	//       but that's tricky, so the cheap approach could be reasonable
	//       and one would rather Advance() than Draw(). Fair enough?
	self.AdvanceGlyph(self.renderer.getGlyphIndex(codePoint))
}

// Advances the Feed's position without drawing anything.
func (self *Feed) AdvanceGlyph(glyphIndex GlyphIndex) {
	self.traverseGlyph(glyphIndex, func(fixed.Point26_6) {})
}

// Advances the Feed's position with a line break.
func (self *Feed) LineBreak() {
	self.Position.X = self.renderer.quantizeX(self.Position.X, self.renderer.direction) // *
	self.Position.Y = self.renderer.quantizeY(self.Position.Y)
	self.Position.Y = self.renderer.applyLineAdvance(self.Position)
	self.Position.X = self.LineBreakX
	self.HasPrevGlyph = false
	// * required because applyLineAdvance may call NotifyFractChange later
}

// Private traverse method used for Draw and Advance.
func (self *Feed) traverseGlyph(glyphIndex GlyphIndex, f func(fixed.Point26_6)) {
	// By spec, we always quantize. While this could be done only if
	// !self.HasPrevGlyph, users may modify the Y position manually and then
	// be bitten by some combinations of caching and quantization modes.
	// While I could also just blame me, I decided to be nicer.
	self.Position.Y = self.renderer.quantizeY(self.Position.Y)

	// create the glyph pair and send it to the proper traverse function
	gpair := glyphPair{ glyphIndex, self.PrevGlyphIndex, self.HasPrevGlyph }
	switch self.renderer.direction {
	case RightToLeft:
		self.Position = self.renderer.traverseGlyphRTL(self.Position, gpair, f)
	case LeftToRight:
		self.Position = self.renderer.traverseGlyphLTR(self.Position, gpair, f)
	default:
		panic("unhandled switch case")
	}

	self.HasPrevGlyph   = true
	self.PrevGlyphIndex = glyphIndex
}
