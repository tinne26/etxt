package fract

import "image"

type Rect struct {
	Min Point
	Max Point
}

func UnitsToRect(minX, minY, maxX, maxY Unit) Rect {
	return Rect{
		Min: Point{ X: minX, Y: minY },
		Max: Point{ X: maxX, Y: maxY },
	}
}

func PointsToRect(min, max Point) Rect {
	return Rect{ Min: min, Max: max }
}

func (self Rect) ImageRect() image.Rectangle {
	minX, minY, maxX, maxY := self.ToInts()
	return image.Rect(minX, minY, maxX, maxY)
}
	
func (self Rect) ToInts() (int, int, int, int) {
	return self.Min.X.ToIntFloor(), self.Min.Y.ToIntFloor(), self.Max.X.ToIntCeil(), self.Max.Y.ToIntCeil()
}

func (self Rect) ToFloat64s() (minX, minY, maxX, maxY float64) {
	return self.Min.X.ToFloat64(), self.Min.Y.ToFloat64(), self.Max.X.ToFloat64(), self.Max.Y.ToFloat64()
}

func (self Rect) Width() Unit {
	return self.Max.X - self.Min.X
}

func (self Rect) IntWidth() int {
	return self.Width().ToIntCeil()
}

func (self Rect) Height() Unit {
	return self.Max.Y - self.Min.Y
}

func (self Rect) IntHeight() int {
	return self.Height().ToIntCeil()
}

func (self Rect) IntOrigin() (int, int) {
	return self.Min.X.ToIntFloor(), self.Min.Y.ToIntFloor()
}

func (self Rect) Empty() bool {
	return self.Min.X >= self.Max.X || self.Min.Y >= self.Max.Y
}

func (self Rect) AddUnits(x, y Unit) Rect {
	self.Min.X += x
	self.Min.Y += y
	self.Max.X += x
	self.Max.Y += y
	return self
}

func (self Rect) AddPoint(pt Point) Rect {
	return self.AddUnits(pt.X, pt.Y)
}

func (self Rect) Contains(pt Point) bool {
	return pt.In(self)
}

func (self Rect) String() string {
	return self.Min.String() + "-" + self.Max.String()
}
