package fract

import "image"
import "strconv"

type Point struct {
	X Unit
	Y Unit
}

func UnitsToPoint(x, y Unit) Point {
	return Point{ X: x, Y: y }
}

func (self Point) ImagePoint() image.Point {
	x, y := self.ToInts()
	return image.Pt(x, y)
}

func (self Point) ToInts() (int, int) {
	return self.X.ToInt(), self.Y.ToInt()
}

func (self Point) ToFloat64s() (x, y float64) {
	return self.X.ToFloat64(), self.Y.ToFloat64()
}

func (self Point) AddUnits(x, y Unit) Point {
	self.X += x
	self.Y += y
	return self
}

func (self Point) AddPoint(pt Point) Point {
	self.X += pt.X
	self.Y += pt.Y
	return self
}

func (self Point) In(rect Rect) bool {
	return self.X >= rect.Min.X && self.X < rect.Max.X && self.Y >= rect.Min.Y && self.Y < rect.Max.Y
}

func (self Point) String() string {
	x := strconv.FormatFloat(self.X.ToFloat64(), 'f', -1, 64)
	y := strconv.FormatFloat(self.Y.ToFloat64(), 'f', -1, 64)
	return "(" + x + ", " + y + ")"
}
