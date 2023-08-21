package etxt

import "github.com/tinne26/etxt/fract"

// notes regarding problems with DrawWithWrap when it 
// comes to twines (wrapping not releasing on v0.0.9):
// - infinite loops may be entered if an effect always pads a consistent
//   amount of space at the start of the line, but the wrap is shorter
// - if multiple effects have to be popped, some may go forward and
//   others may go backwards, so it's unclear if we should break a line
//   after the first, after the whole batch, etc.

func (self *Renderer) complexDrawTwine(target Target, twine Twine, x, y int) {
	self.fractDrawTwine(target, twine, fract.FromInt(x), fract.FromInt(y))
}

func (self *Renderer) fractDrawTwine(target Target, twine Twine, x, y fract.Unit) {
	// preconditions
	if target == nil { panic("can't draw on nil Target") }
	if self.state.align != (Baseline | Left) {
		panic("wip, only (Baseline | Left) align allowed")
	}
	if self.state.textDirection != LeftToRight {
		panic("wip, only LeftToRight direction allowed")
	}

	// return directly on superfluous invocations
	bounds := target.Bounds()
	if bounds.Empty() { return }

	// adjust Y position
	horzQuant, vertQuant := self.fractGetQuantization()
	lineHeight := self.getOpLineHeight()
	y = y.QuantizeUp(vertQuant)

	// skip non-visible portions of the text in the target
	var lineBreakNth int = -1
	maxBaselineY := fract.FromInt(bounds.Max.Y) + lineHeight
	// TODO, unclear if worth it here

	// subdelegate to relevant draw function
	memoState := self.state
	self.fractDrawTwineLeftLTR(target, twine, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
	self.setState(memoState)
	self.twineStorage = self.twineStorage[ : 0]
}

// Precondition: x and y are already quantized.
func (self *Renderer) fractDrawTwineLeftLTR(target Target, twine Twine, lineBreakNth int, x, y, maxY fract.Unit) {
	operator := newTwineOperator(self, twine)
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	var iterator ltrTwineIterator
	for {
		glyphIndex, codePoint := iterator.Next(twine)
		if codePoint == -1 { break }
		if codePoint == rune(twineCcBegin) {
			operator, position, iv = iterator.ProcessCC(target, self, operator, position, iv)
			continue
		}

		if codePoint == '\n' {
			iv.increaseLineBreakNth()
			position = self.advanceLine(position, x, iv.lineBreakNth)
			if position.Y > maxY { break }
		} else {
			if codePoint != -2 {
				glyphIndex = self.getGlyphIndex(self.state.activeFont, codePoint)
			}
			position, iv = self.drawGlyphLTR(target, position, glyphIndex, iv)
		}
	}
	
	// last popAll may be redundant in a lot of cases, but some graphical
	// effects require the pop to trigger anyway, so we can't skip this
	position.X = operator.PopAll(self, target, position.X)
}
