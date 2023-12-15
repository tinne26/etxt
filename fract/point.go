package fract

import "image"
import "strconv"

// A pair of [Unit] coordinates. Commonly used during rendering
// processes to keep track of the pen position within the rendering
// target.
type Point struct {
	X Unit
	Y Unit
}

// Creates a point from a pair of units.
func UnitsToPoint(x, y Unit) Point {
	return Point{ X: x, Y: y }
}

// Creates a point from a pair of ints.
func IntsToPoint(x, y int) Point {
	return Point{ X: FromInt(x), Y: FromInt(y) }
}

// Converts the point coordinates to ints and returns
// them as an [image.Point] stdlib value. The conversion
// will round the units if necessary, which could be
// problematic in some contexts.
func (self Point) ImagePoint() image.Point {
	x, y := self.ToInts()
	return image.Pt(x, y)
}

// Returns the point coordinates as a pair of ints.
// The conversion will round the units if necessary, which
// could be problematic in some contexts.
func (self Point) ToInts() (int, int) {
	return self.X.ToInt(), self.Y.ToInt()
}

// Returns the point coordinates as a pair of float64s.
func (self Point) ToFloat64s() (x, y float64) {
	return self.X.ToFloat64(), self.Y.ToFloat64()
}

// Returns the point coordinates as a pair of float32s.
func (self Point) ToFloat32s() (x, y float32) {
	return self.X.ToFloat32(), self.Y.ToFloat32()
}

// Returns the result of adding the given pair of units to
// the current point coordinates.
func (self Point) AddUnits(x, y Unit) Point {
	self.X += x
	self.Y += y
	return self
}

// Returns the result of adding the two points.
func (self Point) AddPoint(point Point) Point {
	self.X += point.X
	self.Y += point.Y
	return self
}

// Returns whether the current point is inside the given [Rect].
func (self Point) In(rect Rect) bool {
	return self.X >= rect.Min.X && self.X < rect.Max.X && self.Y >= rect.Min.Y && self.Y < rect.Max.Y
}

// Returns a textual representation of the point (e.g.: "(2.5, -4)").
func (self Point) String() string {
	x := strconv.FormatFloat(self.X.ToFloat64(), 'f', -1, 64)
	y := strconv.FormatFloat(self.Y.ToFloat64(), 'f', -1, 64)
	return "(" + x + ", " + y + ")"
}
