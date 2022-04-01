package etxt

import "golang.org/x/image/math/fixed"

// Only private methods related to alignment operations.

// precondition: non-empty text
func (self *Renderer) alignTextDotY(text string, y fixed.Int26_6) fixed.Int26_6 {
	if self.vertAlignRequiresHeight() { // compute height if needed
		return self.alignDotY(self.textHeight(text), y)
	}
	return self.alignDotY(0, y) // height not needed
}

// precondition: non-empty glyphIndices
func (self *Renderer) alignGlyphsDotY(y fixed.Int26_6) fixed.Int26_6 {
	if self.vertAlignRequiresHeight() { // compute height if needed
		if self.metrics == nil { self.updateMetrics() }
		return self.alignDotY(self.metrics.Height, y)
	}
	return self.alignDotY(0, y) // height not needed
}

// Given a reference y coordinate, it aligns it to the baseline drawing
// point based on the current vertical align and the height of the
// content to draw.
func (self *Renderer) alignDotY(height, y fixed.Int26_6) fixed.Int26_6 {
	// return early on simple Baseline case
	if self.vertAlign == Baseline { return y }

	// align y coordinate
	if self.metrics == nil { self.updateMetrics() }
	switch self.vertAlign {
	case YCenter:
		y += self.metrics.Ascent
		if self.lineSpacing < 0 { // evil edge case
			y -= height - self.metrics.Height
		}
		y -= (height >> 1)
	case Top:
		y += self.metrics.Ascent
		if self.lineSpacing < 0 { // evil edge case
			y -= height - self.metrics.Height
		}
	case Bottom:
		y -= self.metrics.Descent
		if self.lineSpacing >= 0 {
			y -= height - self.metrics.Height
		} else { // evil edge case
			y += height - self.metrics.Height
		}
	}

	return y
}

func (self *Renderer) vertAlignRequiresHeight() bool {
	switch self.vertAlign {
	case YCenter  : return true
	case Bottom   : return true
	case Top      : return (self.lineSpacing < 0)
	case Baseline : return false
	default:
		panic("unhandled switch case")
	}
}
