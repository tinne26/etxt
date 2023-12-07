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

	// return directly on superfluous invocations
	bounds := target.Bounds()
	if bounds.Empty() { return }

	// adjust Y position
	horzQuant, vertQuant := self.fractGetQuantization()
	lineHeight := self.getOpLineHeight()
	vertAlign := self.state.align.Vert()
	switch vertAlign {
	case Top:
		y = (y + self.getOpAscent()).QuantizeUp(vertQuant)
	case CapLine:
		capHeight := self.getSlowOpCapHeight()
		y = (y + capHeight).QuantizeUp(vertQuant)
	case Midline:
		xheight := self.getSlowOpXHeight()
		y = (y + xheight).QuantizeUp(vertQuant)
	case VertCenter:
		heightSizer := getTwineHeightSizer()
		heightSizer.Initialize(self, twine)
		height := heightSizer.Measure(self, target)
		releaseTwineHeightSizer(heightSizer)
		y = (y + self.getOpAscent() - (height >> 1)).QuantizeUp(vertQuant)
	case Baseline:
		y = y.QuantizeUp(vertQuant)
	case LastBaseline:
		heightSizer := getTwineHeightSizer()
		heightSizer.Initialize(self, twine)
		height := heightSizer.Measure(self, target)
		releaseTwineHeightSizer(heightSizer)
		qtLineHeight := lineHeight.QuantizeUp(vertQuant)
		if height >= qtLineHeight { height -= qtLineHeight }
		y = (y - height).QuantizeUp(vertQuant)
	case Bottom:
		heightSizer := getTwineHeightSizer()
		heightSizer.Initialize(self, twine)
		height := heightSizer.Measure(self, target)
		releaseTwineHeightSizer(heightSizer)
		y = (y + self.getOpAscent() - height).QuantizeUp(vertQuant)
	default:
		panic(vertAlign)
	}

	// skip non-visible portions of the text in the target
	var lineBreakNth int = -1
	maxBaselineY := fract.FromInt(bounds.Max.Y) + lineHeight
	// TODO: unclear if worth it here. it also messes with
	//       pushes and pops. But for centered and counter
	//       dir, it should be reasonable enough. But then
	//       I have to pass the operator right away. hmmm.

	// subdelegate to relevant draw function
	switch self.state.align.Horz() {
	case Left:
		if self.state.textDirection == LeftToRight {
			self.fractDrawTwineAlignMatchDir(target, twine, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
		} else {
			self.fractDrawTwineLeftRTL(target, twine, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
		}
	case Right:
		if self.state.textDirection == LeftToRight {
			self.fractDrawTwineRightLTR(target, twine, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
		} else {
			self.fractDrawTwineAlignMatchDir(target, twine, lineBreakNth, x.QuantizeUp(horzQuant), y, maxBaselineY)
		}
	case HorzCenter:
		if self.state.textDirection == LeftToRight {
			self.fractDrawTwineCenterLTR(target, twine, lineBreakNth, x, y, maxBaselineY)
		} else {
			self.fractDrawTwineCenterRTL(target, twine, lineBreakNth, x, y, maxBaselineY)
		}
	default:
		panic(self.state.align.Horz())
	}
	self.twineStorage = self.twineStorage[ : 0]
}

// Precondition: x and y are already quantized.
// TODO: I might have to provide an optimization on/off flag. In regular draws I don't think realistic
// cases where the optimization breaks anything exist, but for the case of twines... it's another story.
// I think the cases are still rather extreme, but it could more realistically be a thing. I can always
// add a new CC afterwards though.
func (self *Renderer) fractDrawTwineAlignMatchDir(target Target, twine Twine, lineBreakNth int, x, y, maxY fract.Unit) {
	operator := getTwineOperator()
	var mustDraw bool = true
	operator.Initialize(self, twine, mustDraw, maxY)
	operator.SetDefaultNewLineX(x)
	
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		// notify first fractX shift position, which will be used.
		// during centering, we have to call this manually later
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
			position, iv, _ = operator.LineBreak(self, target, position, iv)
			if position.Y > maxY { break loop }
		case rune(twineCcBegin):
			position, iv = operator.ProcessCC(self, target, position, iv)
		case twineRuneNotAvailable:
			position, iv = operator.Operate(self, target, position, glyphIndex, iv)
		default:
			glyphIndex = self.getGlyphIndex(self.state.activeFont, codePoint)
			position, iv = operator.Operate(self, target, position, glyphIndex, iv)
		}
	}

	releaseTwineOperator(operator)
}

func (self *Renderer) fractDrawTwineLeftRTL(target Target, twine Twine, lineBreakNth int, x, y, maxY fract.Unit) {
	operator := getTwineLineOperator()
	operator.Initialize(self, twine, maxY)
	operator.SetDefaultNewLineX(x)
	
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	for !operator.Ended() {
		position, iv = operator.MeasureAndDrawLine(self, target, iv, position.Y,
			func(width fract.Unit) fract.Unit {
				lineStartX := x + width
				operator.SetDefaultNewLineX(lineStartX)
				return lineStartX
			})
		if position.Y > maxY { break }
	}

	releaseTwineLineOperator(operator)
}

func (self *Renderer) fractDrawTwineRightLTR(target Target, twine Twine, lineBreakNth int, x, y, maxY fract.Unit) {
	operator := getTwineLineOperator()
	operator.Initialize(self, twine, maxY)
	operator.SetDefaultNewLineX(x)
	
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	for !operator.Ended() {
		position, iv = operator.MeasureAndDrawLine(self, target, iv, position.Y,
			func(width fract.Unit) fract.Unit {
				lineStartX := x - width
				operator.SetDefaultNewLineX(lineStartX)
				return lineStartX
			})
		if position.Y > maxY { break }
	}

	releaseTwineLineOperator(operator)
}

func (self *Renderer) fractDrawTwineCenterLTR(target Target, twine Twine, lineBreakNth int, x, y, maxY fract.Unit) {
	operator := getTwineLineOperator()
	operator.Initialize(self, twine, maxY)
	operator.SetDefaultNewLineX(x)
	
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	for !operator.Ended() {
		position, iv = operator.MeasureAndDrawLine(self, target, iv, position.Y,
			func(width fract.Unit) fract.Unit {
				lineStartX := x - width/2
				operator.SetDefaultNewLineX(lineStartX)
				return lineStartX
			})
		if position.Y > maxY { break }
	}

	releaseTwineLineOperator(operator)
}

func (self *Renderer) fractDrawTwineCenterRTL(target Target, twine Twine, lineBreakNth int, x, y, maxY fract.Unit) {
	operator := getTwineLineOperator()
	operator.Initialize(self, twine, maxY)
	operator.SetDefaultNewLineX(x)
	
	position := fract.UnitsToPoint(x, y)
	var iv drawInternalValues
	iv.prevFractX = position.X.FractShift()
	iv.lineBreakNth = lineBreakNth
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(position)
	}

	for !operator.Ended() {
		position, iv = operator.MeasureAndDrawLine(self, target, iv, position.Y,
			func(width fract.Unit) fract.Unit {
				lineStartX := x + width/2
				operator.SetDefaultNewLineX(lineStartX)
				return lineStartX
			})
		if position.Y > maxY { break }
	}

	releaseTwineLineOperator(operator)
}
