package emask

import "image"
import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt/efixed"

// Given some glyph bounds and a fractional pixel position, it figures out
// what integer size must be used to fit the bounds, what normalization
// offset must be applied to keep the coordinates in the positive plane,
// and what final offset must be applied to the final mask to align its
// bounds to the glyph origin. This is used in NewContour functions.
func figureOutBounds(bounds fixed.Rectangle26_6, fract fixed.Point26_6) (image.Point, fixed.Point26_6, image.Point) {
	floorMinX := efixed.Floor(bounds.Min.X)
	floorMinY := efixed.Floor(bounds.Min.Y)
	var maskCorrection image.Point
	maskCorrection.X = int(floorMinX >> 6)
	maskCorrection.Y = int(floorMinY >> 6)

	var normOffset fixed.Point26_6
	normOffset.X = -floorMinX + fract.X
	normOffset.Y = -floorMinY + fract.Y
	width  := (bounds.Max.X + normOffset.X).Ceil()
	height := (bounds.Max.Y + normOffset.Y).Ceil()
	return image.Pt(width, height), normOffset, maskCorrection
}
