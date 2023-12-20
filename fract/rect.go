package fract

import "image"

// A pair of [Point] values defining a rectangular region.
// Like [image.Rectangle], the Max point is not included
// in the rectangle. The behavior for malformed rectangles
// is undefined.
type Rect struct {
	Min Point
	Max Point
}

// Creates a rect from a set of four units.
func UnitsToRect(minX, minY, maxX, maxY Unit) Rect {
	return Rect{
		Min: Point{ X: minX, Y: minY },
		Max: Point{ X: maxX, Y: maxY },
	}
}

// Creates a rect from a pair of points.
func PointsToRect(min, max Point) Rect {
	return Rect{ Min: min, Max: max }
}

// Creates a rect from a set of four integers.
func IntsToRect(minX, minY, maxX, maxY int) Rect {
	return Rect{
		Min: Point{ X: FromInt(minX), Y: FromInt(minY) },
		Max: Point{ X: FromInt(maxX), Y: FromInt(maxY) },
	}
}

// Creates a rect from an [image.Rectangle].
func FromImageRect(rect image.Rectangle) Rect {
	return IntsToRect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
}

// Converts the rect coordinates to ints and returns
// them as an [image.Rectangle] stdlib value. The conversion
// will round the units if necessary, which could be
// problematic in some contexts.
func (self Rect) ImageRect() image.Rectangle {
	minX, minY, maxX, maxY := self.ToInts()
	return image.Rect(minX, minY, maxX, maxY)
}

// Returns the rect coordinates as a set of four ints.
// The conversion may introduce a loss of precision, but the
// returned ints are guaranteed to contain the original
// rect.
func (self Rect) ToInts() (minX, minY, maxX, maxY int) {
	return self.Min.X.ToIntFloor(), self.Min.Y.ToIntFloor(), self.Max.X.ToIntCeil(), self.Max.Y.ToIntCeil()
}

// Returns the rect coordinates as a set of four float64s.
func (self Rect) ToFloat64s() (minX, minY, maxX, maxY float64) {
	return self.Min.X.ToFloat64(), self.Min.Y.ToFloat64(), self.Max.X.ToFloat64(), self.Max.Y.ToFloat64()
}

// Returns the rect coordinates as a set of four float32s.
func (self Rect) ToFloat32s() (minX, minY, maxX, maxY float32) {
	return self.Min.X.ToFloat32(), self.Min.Y.ToFloat32(), self.Max.X.ToFloat32(), self.Max.Y.ToFloat32()
}

// Returns the width of the rect.
func (self Rect) Width() Unit {
	return self.Max.X - self.Min.X
}

// Returns the width of the rect as an int. The conversion
// may introduce a loss of precision, but the returned
// int width is guaranteed to be >= than the original.
func (self Rect) IntWidth() int {
	return self.Width().ToIntCeil()
}

// Returns the height of the rect.
func (self Rect) Height() Unit {
	return self.Max.Y - self.Min.Y
}

// Returns the height of the rect as an int. The conversion
// may introduce a loss of precision, but the returned
// int height is guaranteed to be >= than the original.
func (self Rect) IntHeight() int {
	return self.Height().ToIntCeil()
}

// Utility method equivalent to ([Rect.Width](), [Rect.Height]()).
func (self Rect) Size() (width, height Unit) {
	return self.Width(), self.Height()
}

// Utility method equivalent to ([Rect.IntWidth](), [Rect.IntHeight]()).
func (self Rect) IntSize() (width, height int) {
	return self.IntWidth(), self.IntHeight()
}

// Returns the Min point as a pair of ints. The conversion
// may introduce a loss of precision, but the returned
// coordinates are guaranteed to be <= than the original.
func (self Rect) IntOrigin() (int, int) {
	return self.Min.X.ToIntFloor(), self.Min.Y.ToIntFloor()
}

// Returns whether the Min point is (0, 0) or not.
func (self Rect) HasZeroOrigin() bool {
	return self.Min.X == 0 && self.Min.Y == 0
}

// Returns whether the rect is empty or not.
func (self Rect) Empty() bool {
	return self.Min.X >= self.Max.X || self.Min.Y >= self.Max.Y
}

// Returns the result of applying the given paddings to each
// side of the rect. In other words, the rect's width after the
// padding is increased by horzPad*2 (likewise for the height
// with vertPad*2).
func (self Rect) PadUnits(horzPad, vertPad Unit) Rect {
	return UnitsToRect(
		self.Min.X - horzPad, self.Min.Y - vertPad,
		self.Max.X + horzPad, self.Max.Y + vertPad,
	)
}

// Same as [Rect.PadUnits]() but with ints.
func (self Rect) PadInts(horzMargin, vertMargin int) Rect {
	return self.PadUnits(FromInt(horzMargin), FromInt(vertMargin))
}

// Returns the result of translating the rect by the given values.
func (self Rect) AddUnits(x, y Unit) Rect {
	self.Min.X += x
	self.Min.Y += y
	self.Max.X += x
	self.Max.Y += y
	return self
}

// Returns the result of translating the rect by the given values.
func (self Rect) AddInts(x, y int) Rect {
	xFract, yFract := FromInt(x), FromInt(y)
	return self.AddUnits(xFract, yFract)
}

// Returns the result of translating the rect by the given value.
func (self Rect) AddImagePoint(point image.Point) Rect {
	return self.AddInts(point.X, point.Y)
}

// Returns the result of translating the rect by the given value.
func (self Rect) AddPoint(pt Point) Rect {
	return self.AddUnits(pt.X, pt.Y)
}

// Returns the result of translating the rect so its center
// becomes aligned to the given coordinates.
func (self Rect) CenteredAtIntCoords(x, y int) Rect {
	ux, uy := FromInt(x), FromInt(y)
	hw, hh := self.Width() >> 1, self.Height() >> 1
	return Rect{
		Min: Point{ X: ux - hw, Y: uy - hh },
		Max: Point{ X: ux + hw, Y: uy + hh },
	}
}

// Returns whether the rect contains the given point or not.
//
// Remember that point == Rect.Min is included, but point == Rect.Max
// is not.
func (self Rect) Contains(point Point) bool {
	return point.In(self)
}

// Returns a textual representation of the rect (e.g.: "(0, 0) - (1.5, 8.5)").
func (self Rect) String() string {
	return self.Min.String() + "-" + self.Max.String()
}
