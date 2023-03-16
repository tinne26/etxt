package fract

import "testing"

func TestPoint(t *testing.T) {
	point := UnitsToPoint(64, 31)
	imgPt := point.ImagePoint()
	if imgPt.X != 1 || imgPt.Y != 0 {
		t.Fatalf("expected (X: 1, Y: 0), got %v", imgPt)
	}
	point = point.AddUnits(0, 1)
	imgPt = point.ImagePoint()
	if imgPt.X != 1 || imgPt.Y != 1 {
		t.Fatalf("expected (X: 1, Y: 1), got %v", imgPt)
	}

	if point.String() != "(1, 0.5)" {
		t.Fatalf("expected (1, 0.5), got %s", point.String())
	}
	point = point.AddPoint(point)
	if point.String() != "(2, 1)" {
		t.Fatalf("expected (2, 1), got %s", point.String())
	}
	x, y := point.ToFloat64s()
	if x != 2 || y != 1 {
		t.Fatalf("expected (2, 1), got (%f, %f)", x, y)
	}

	if !point.In(UnitsToRect(128, 64, 129, 65)) {
		t.Fatalf("point.In(rect) #1: expected inside, got outside")
	}
	if point.In(UnitsToRect(128, 64, 129, 64)) {
		t.Fatalf("point.In(rect) #2: expected outside, got inside")
	}
	if point.In(UnitsToRect(128, 64, 128, 65)) {
		t.Fatalf("point.In(rect) #3: expected outside, got inside")
	}
	if point.In(UnitsToRect(0, 0, 64, 64)) {
		t.Fatalf("point.In(rect) #4: expected outside, got inside")
	}
}
