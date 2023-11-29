package etxt

import "github.com/tinne26/etxt/fract"

// notes regarding problems with DrawWithWrap when it 
// comes to twines (wrapping not releasing on v0.0.9):
// - infinite loops may be entered if an effect always pads a consistent
//   amount of space at the start of the line, but the wrap is shorter
// - if multiple effects have to be popped, some may go forward and
//   others may go backwards, so it's unclear if we should break a line
//   after the first, after the whole batch, etc.

func (self *Renderer) twineDraw(target Target, twine Twine, x, y int) {
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
	// TODO, unclear if worth it here, it can get quite crazy
	// with accumulated effects and so on.

	// subdelegate to relevant draw function
	memoState := self.state
	self.fractDrawTwineLeftLTR(target, twine, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
	self.setState(memoState)
	self.twineStorage = self.twineStorage[ : 0]
}

// Precondition: x and y are already quantized.
// TODO: I might have to provide an optimization on/off flag. In regular draws I don't think realistic
// cases where the optimization breaks anything exist, but for the case of twines... it's another story.
// I think the cases are still rather extreme, but it could more realistically be a thing. I can always
// add a new CC afterwards though.
func (self *Renderer) fractDrawTwineLeftLTR(target Target, twine Twine, lineBreakNth int, x, y, maxY fract.Unit) {
	operator := getTwineOperator()
	var mustDraw bool = true
	operator.Initialize(self, twine, mustDraw, x, maxY)
	
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

loop:
	for {
		codePoint, glyphIndex := operator.Next()
		switch codePoint {
		case twineRuneEndOfText:
			var dpReset bool
			position, iv, dpReset = operator.PopAll(self, target, position, iv)
			if dpReset { continue }
			break loop
		case '\n':
			position, iv = operator.LineBreak(self, target, position, iv)
			if position.Y > maxY { break loop }
		case rune(twineCcBegin):
			position, iv = operator.ProcessCC(self, target, position, iv)
		case twineRuneNotAvailable:
			position, iv = operator.OperateLTR(self, target, position, glyphIndex, iv)
		default:
			glyphIndex = self.getGlyphIndex(self.state.activeFont, codePoint)
			position, iv = operator.OperateLTR(self, target, position, glyphIndex, iv)
		}
	}

	releaseTwineOperator(operator)
}
