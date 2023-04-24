//go:build nope

package etxt

import "image"

import "github.com/tinne26/etxt/fract"

// Used by [Renderer.DrawInRect](), [Renderer.MeasureWithWrap]() and
// similar functions that operate within a delimited rectangular area.
type RectOptions struct {
	// Coordinates and dimensions of the area that we want to
	// draw the text in. See [RectOptions.SetArea]() if you
	// want to set the area with an image.Rectangle.
	Area fract.Rect

	// Line break mode. Defaults to [LineOverflow].
	LineBreak RectLineBreak
	
	// When set to true, this flag will prevent the
	// renderer from drawing lines that may not fully
	// fall within the Area's allotted vertical space.
	VertClip bool
	
	// If KnuthPlass is ever added...
	//IdealMargin uint32
}

// Utility method to set the rect area with [image.Rectangle] 
// instead of [fract.Rectangle].
func (self *RectOptions) SetArea(rect image.Rectangle) {
	minX, minY := fract.FromInt(rect.Min.X), fract.FromInt(rect.Min.Y)
	maxX, maxY := fract.FromInt(rect.Max.X), fract.FromInt(rect.Max.Y)
	self.Area = fract.UnitsToRect(minX, minY, maxX, maxY)
}

// Line breaking modes available for [RectOptions].
type RectLineBreak uint8
const (
	LineOverflow   RectLineBreak = 0
	LineClipLetter RectLineBreak = 1 // overflowing letters won't be drawn
	LineClipWord   RectLineBreak = 2 // overflowing words won't be drawn
	LineEllipsis   RectLineBreak = 3 // overflowing fragments will get a "..." ending
	LineWrapGreedy RectLineBreak = 4 // overflowing fragments will go to the next line
	//LineWrapKnuthPlass RectLineBreak = 5 // maybe someday
)
