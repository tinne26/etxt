package etxt

import "github.com/tinne26/etxt/fract"

// see the regular fractMeasure() function for extra documentation
func (self *Renderer) twineMeasure(twine Twine) fract.Rect {
	operator := getTwineLineOperator()
	operator.Initialize(self, twine, 0x7FFFFFFF)
	
	var position fract.Point
	var iv drawInternalValues
	var width, lineWidth fract.Unit
	var lineBreaksOnly bool = true
	vertQuant := fract.Unit(self.state.vertQuantization)

	for !operator.Ended() {
		var breakRune rune
		lineWidth, iv, breakRune = operator.MeasureAndAdvanceLine(self, nil, iv, position.Y)
		if lineWidth > width {
			width = lineWidth
			if lineBreaksOnly {
				lineBreaksOnly = false
				position.Y += self.getOpLineHeight()
				position.Y  = position.Y.QuantizeUp(vertQuant)
			}
		}
		if breakRune != '\n' { break } // twineRuneEndOfText
		position.X = lineWidth
		position, iv = operator.AdvanceLineBreak(self, nil, position, iv)
	}

	releaseTwineLineOperator(operator)

	width = width.QuantizeUp(fract.Unit(self.state.horzQuantization))
	return fract.Rect{ Max: fract.UnitsToPoint(width, position.Y) }
}
