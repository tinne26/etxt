package etxt

import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt/efixed"

// Returns a Feed object linked to the Renderer.
//
// Feeds are the lowest level mechanism to draw text in etxt, as they
// expose and allow to modify the drawing position manually.
//
// Unlike Traverse* methods, though, Feeds can't automatically align
// text because the content to work with isn't known ahead of time.
// Only vertical align will be applied to the starting position as if
// the content had a single line.
func (self *Renderer) NewFeed(xy fixed.Point26_6) *Feed {
	xy.Y = self.alignGlyphsDotY(xy.Y)
	return &Feed { renderer: self, Position: xy, LineBreakX: xy.X }
}

// During traversal processes, having both the current and previous
// glyph is important in order to be able to apply kerning.
type glyphPair struct {
	// Current glyph index.
	CurrentIndex GlyphIndex

	// Previous glyph index.
	PreviousIndex GlyphIndex

	// Whether the PreviousIndex is valid and can be used
	// for purposes like kerning.
	HasPrevious bool
}

// Low-level method that can be used to implement your own drawing and
// bounding operations.
//
// The given function is called for each character in the input string.
// On line breaks, the function is called with '\n' and glyph index 0.
//
// The returned coordinates correspond to the baseline position of the
// next glyph that would have to be drawn if the input was longer. The
// coordinates are unquantized and haven't had kerning applied (as the
// next glyph is not known yet). Notice that align and text direction
// can affect the returned coordinate:
//  - If the horizontal align is etxt.Left, the returned coordinate will
//    be on the right side of the last character drawn.
//  - If the horizontal align is etxt.Right, the returned coordinate will
//    be on the left side of the last character drawn.
//  - If the horizontal align is etxt.XCenter, the returned coordinate will
//    be on the right side of the last character drawn if the text direction
//    is LeftToRight, or on the left side otherwise.
//
// This returned coordinate can be useful when implementing bidirectional
// text renderers, custom multi-style renderers and similar, though for
// heterogeneous styling using a Feed is often more appropriate.
func (self *Renderer) Traverse(text string, xy fixed.Point26_6, operation func(fixed.Point26_6, rune, GlyphIndex)) fixed.Point26_6 {
	// NOTE: the spec says the returned coordinates are unquantized,
	//       but the y will be quantized if another character has been
	//       written earlier in the same line. So, it's more like
	//       unquantized relative to the last position. This is expected
	//       behavior, and I doubt anyone will care, but at least it's
	//       written down.

	if text == "" { return xy } // empty case

	// prepare helper variables
	hasPrevGlyph  := false
	previousIndex := GlyphIndex(0)
	dir, traverseInReverse := self.traversalMode()
	traverseFunc := self.getTraverseFunc(dir)
	xy.Y = self.alignTextDotY(text, xy.Y)
	lineHorzResetPoint := xy.X

	// create iterator and adjust for centered align
	iterator := newStrIterator(text, traverseInReverse)
	if self.horzAlign == XCenter {
		xy.X = self.centerLineStringX(iterator, lineHorzResetPoint, dir)
	}
	dot := xy
	self.preFractPositionNotify(dot)

	// iterate text code points
	for {
		codePoint := iterator.Next()
		if codePoint == -1 { return dot } // end condition

		// handle special line break case by moving the dot
		if codePoint == '\n' {
			dot.X = self.quantizeX(dot.X, dir)
			if !hasPrevGlyph { dot.Y = self.quantizeY(dot.Y) }
			operation(dot, '\n', 0)
			dot.Y = self.applyLineAdvance(dot)
			if self.horzAlign == XCenter {
				dot.X = self.centerLineStringX(iterator, lineHorzResetPoint, dir)
			} else {
				dot.X = lineHorzResetPoint
			}
			hasPrevGlyph = false
			continue
		}

		// quantize now (subtle consistency concerns)
		if !hasPrevGlyph {
			dot.X = self.quantizeX(dot.X, dir)
			dot.Y = self.quantizeY(dot.Y)
		}

		// get the glyph index for the current character and traverse it
		index := self.getGlyphIndex(codePoint)
		dot = traverseFunc(dot, glyphPair{ index, previousIndex, hasPrevGlyph },
			func(dot fixed.Point26_6) { operation(dot, codePoint, index) })
		hasPrevGlyph = true
		previousIndex = index
	}
}

// Like Traverse, but for glyph indices.
//
// Since glyph indices are virtually only used when doing [text shaping]
// —and that's a complex process on itself— I decided to omit most other
// simple glyph functions in order to keep the API cleaner. If you are
// already working with glyphs, implementing your own operations on top
// of TraverseGlyphs should be fairly simple.
//
// [text shaping]: https://github.com/tinne26/etxt/blob/main/docs/shaping.md
func (self *Renderer) TraverseGlyphs(glyphIndices []GlyphIndex, xy fixed.Point26_6, operation func(fixed.Point26_6, GlyphIndex)) fixed.Point26_6 {
	if len(glyphIndices) == 0 { return xy } // empty case

	// prepare helper variables, aligns, etc
	previousIndex := GlyphIndex(0)
	dir, traverseInReverse := self.traversalMode()
	traverseFunc := self.getTraverseFunc(dir)
	xy.Y = self.alignGlyphsDotY(xy.Y)
	if self.horzAlign == XCenter { // consider xcenter align
		hw := (self.SelectionRectGlyphs(glyphIndices).Width >> 1)
		if dir == LeftToRight { xy.X -= hw } else { xy.X += hw }
	}
	self.preFractPositionNotify(xy)
	xy.X = self.quantizeX(xy.X, dir)
	xy.Y = self.quantizeY(xy.Y)
	dot := xy

	// iterate first glyph (with prevGlyph == false)
	iterator := newGlyphsIterator(glyphIndices, traverseInReverse)
	index, _ := iterator.Next()
	dot = traverseFunc(dot, glyphPair{ index, previousIndex, false },
		func(dot fixed.Point26_6) { operation(dot, index)} )
	previousIndex = index

	// iterate all remaining glyphs
	for {
		index, done := iterator.Next()
		if done { return dot }
		dot = traverseFunc(dot, glyphPair{ index, previousIndex, true },
			func(dot fixed.Point26_6) { operation(dot, index)} )
		previousIndex = index
	}
}

// --- helper methods ---

func (self *Renderer) quantizeY(y fixed.Int26_6) fixed.Int26_6 {
	if self.quantization == QuantizeNone { return y }
	if self.GetLineAdvance() >= 0 {
		return efixed.RoundHalfUp(y)
	} else {
		return efixed.RoundHalfDown(y)
	}
}

func (self *Renderer) quantizeX(x fixed.Int26_6, dir Direction) fixed.Int26_6 {
	if self.quantization == QuantizeNone { return x }
	if dir == LeftToRight {
		return efixed.RoundHalfUp(x)
	} else { // RightToLeft
		return efixed.RoundHalfDown(x)
	}
}

func (self *Renderer) centerLineStringX(iterator strIterator, lineHorzResetPoint fixed.Int26_6, dir Direction) fixed.Int26_6 {
	line := iterator.UntilNextLineBreak()
	halfWidth := (self.SelectionRect(line).Width >> 1)
	if dir == LeftToRight {
		return lineHorzResetPoint - halfWidth
	} else {
		return lineHorzResetPoint + halfWidth
	}
}

// Notify fractional position change at the start of traversal.
func (self *Renderer) preFractPositionNotify(dot fixed.Point26_6) {
	if self.cacheHandler != nil {
		if self.quantization == QuantizeNone {
			self.cacheHandler.NotifyFractChange(dot) // only required for Y
		} else {
			// X is expected to be notified later too, so
			// this works for QuantizeVert without issue
			self.cacheHandler.NotifyFractChange(fixed.Point26_6{})
		}
	}
}

func (self *Renderer) getGlyphIndex(codePoint rune) GlyphIndex {
	index, err := self.font.GlyphIndex(&self.buffer, codePoint)
	if err != nil { panic("font.GlyphIndex error: " + err.Error()) }
	if index == 0 {
		msg := "glyph index for '" + string(codePoint) + "' ["
		msg += runeToUnicodeCode(codePoint) + "] missing"
		panic(msg)
	}
	return index
}

// The returned bool is true when the traversal needs to be done
// in reverse (from last line character to first).
func (self *Renderer) traversalMode() (Direction, bool) {
	switch self.horzAlign {
	case Left  : return LeftToRight, (self.direction == RightToLeft)
	case Right : return RightToLeft, (self.direction == LeftToRight)
	}
	return self.direction, false
}

type traverseFuncType func(fixed.Point26_6, glyphPair, func(fixed.Point26_6)) fixed.Point26_6
func (self *Renderer) getTraverseFunc(dir Direction) traverseFuncType {
	if dir == LeftToRight { return self.traverseGlyphLTR }
	return self.traverseGlyphRTL
}

func (self *Renderer) traverseGlyphLTR(dot fixed.Point26_6, glyphSeq glyphPair, operation func(fixed.Point26_6)) fixed.Point26_6 {
	// kern
	if glyphSeq.HasPrevious { // apply kerning
		prev, curr := glyphSeq.PreviousIndex, glyphSeq.CurrentIndex
		dot.X += self.sizer.Kern(self.font, prev, curr, self.sizePx)
		if self.quantization == QuantizeFull {
			dot.X = efixed.RoundHalfUp(dot.X)
		} else if self.cacheHandler != nil {
			self.cacheHandler.NotifyFractChange(dot)
		}
	}

	// operate
	operation(dot)

	// advance
	dot.X += self.sizer.Advance(self.font, glyphSeq.CurrentIndex, self.sizePx)
	return dot
}

func (self *Renderer) traverseGlyphRTL(dot fixed.Point26_6, glyphSeq glyphPair, operation func(fixed.Point26_6)) fixed.Point26_6 {
	// advance
	dot.X -= self.sizer.Advance(self.font, glyphSeq.CurrentIndex, self.sizePx)

	// kern and pad
	if glyphSeq.HasPrevious {
		prev, curr := glyphSeq.PreviousIndex, glyphSeq.CurrentIndex
		dot.X -= self.sizer.Kern(self.font, curr, prev, self.sizePx)
	}

	// quantize position
	if self.quantization == QuantizeFull {
		dot.X = efixed.RoundHalfDown(dot.X)
	} else if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(dot)
	}

	// operate
	operation(dot)
	return dot
}

// Apply line advance to the given coordinate applying quantization if
// relevant, notifying the cache handler fractional pixel change, etc.
func (self *Renderer) applyLineAdvance(dot fixed.Point26_6) fixed.Int26_6 {
	// handle non-quantized case (notifying fractional position change)
	if self.quantization == QuantizeNone {
		dot.Y += self.GetLineAdvance()
		if self.cacheHandler != nil {
			self.cacheHandler.NotifyFractChange(dot)
		}
		return dot.Y
	}

	// handle quantized case (round but don't notify fractional change)
	lineAdvance := self.GetLineAdvance()
	if lineAdvance >= 0 {
		return efixed.RoundHalfUp(dot.Y + lineAdvance)
	} else {
		return efixed.RoundHalfDown(dot.Y + lineAdvance)
	}
}
