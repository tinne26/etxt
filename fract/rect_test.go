package fract

import "testing"

func TestRectTrivial(t *testing.T) {
	rect := UnitsToRect(0, 0, 64, 64)
	if rect.Width() != 64 {
		t.Fatal("incorrect width")
	}
	if rect.Height() != 64 {
		t.Fatal("incorrect width")
	}

	imgRect := rect.ImageRect()
	if imgRect.Min.X != 0 || imgRect.Min.Y != 0 || imgRect.Max.X != 1 || imgRect.Max.Y != 1 {
		t.Fatal("incorrect ImageRect() conversion")
	}

	if imgRect.Dx() != rect.IntWidth() || imgRect.Dy() != rect.IntHeight() {
		t.Fatal("discordance between int width/heigth and ImageRect() width/height")
	}

	if rect.Empty() {
		t.Fatal("should not be empty")
	}

	rect = rect.AddUnits(32, 32)
	minx, miny, maxx, maxy := rect.ToFloat64s()
	if minx != 0.5 || miny != 0.5 || maxx != 1.5 || maxy != 1.5 {
		t.Fatal("invalid ToFloat64s() conversion")
	}

	imgRect = rect.ImageRect()
	if imgRect.Dx() != 2 || imgRect.Dy() != 2 {
		t.Fatal("expected width and height to be 2")
	}
	if rect.IntWidth() != 1 || rect.IntHeight() != 1 {
		t.Fatal("expected precise width and height here")
	}
	ox, oy := rect.IntOrigin()
	if ox != 0 || oy != 0 {
		t.Fatal("expected (0, 0) origin")
	}
}

func TestRectPoints(t *testing.T) {
	pt1, pt2 := UnitsToPoint(33, 33), UnitsToPoint(36, 36)
	rect := PointsToRect(pt2, pt1)
	if !rect.Empty() {
		t.Fatal("expected empty rect")
	}
	rect = PointsToRect(pt1, pt2)
	if rect.String() != pt1.String()+"-"+pt2.String() {
		t.Fatalf("unexpected rect.String() value '%s'", rect.String())
	}

	if !rect.Contains(pt1) {
		t.Fatal("expected pt1 to be contained")
	}
	if rect.Contains(pt2) {
		t.Fatal("expected pt2 to NOT be contained")
	}

	rect = rect.AddPoint(pt1)
	if !rect.Contains(pt1.AddPoint(pt1)) {
		t.Fatal("expected pt1 + pt1 to be contained")
	}
	if rect.Contains(pt1.AddPoint(pt2)) {
		t.Fatal("expected pt1 + pt2 to NOT be contained")
	}
}
