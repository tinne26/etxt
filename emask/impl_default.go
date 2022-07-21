package emask

import "image"
import "image/draw"

import "golang.org/x/image/vector"
import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

// The DefaultRasterizer is a wrapper to make [golang.org/x/image/vector.Rasterizer]
// conform to the [Rasterizer] interface.
type DefaultRasterizer struct {
	rasterizer vector.Rasterizer
	rectOffset image.Point // offset to align the final mask rect to the bounds
	normOffset fixed.Point26_6 // offset to normalize points to the positive
	                           // quadrant starting from the fractional coords

	cacheSignature uint64
	onChange func(Rasterizer)

	// Notice that the x/image/vector rasterizer expects coords in the
	// positive quadrant, which is why we need so many offsets here.
}

// Satisfies the [UserCfgCacheSignature] interface.
func (self *DefaultRasterizer) SetHighByte(value uint8) {
	self.cacheSignature = uint64(value) << 56
	if self.onChange != nil { self.onChange(self) }
}

// Satisfies the [Rasterizer] interface.
func (self *DefaultRasterizer) SetOnChangeFunc(onChange func(Rasterizer)) {
	self.onChange = onChange
}

// Satisfies the [Rasterizer] interface.
func (self *DefaultRasterizer) CacheSignature() uint64 {
	return self.cacheSignature
}

// Moves the current position to the given point.
func (self *DefaultRasterizer) MoveTo(point fixed.Point26_6) {
	x, y := self.fixedToFloat32Coords(point)
	self.rasterizer.MoveTo(x, y)
}

// Creates a straight boundary from the current position to the given point.
func (self *DefaultRasterizer) LineTo(point fixed.Point26_6) {
	x, y := self.fixedToFloat32Coords(point)
	self.rasterizer.LineTo(x, y)
}

// Creates a quadratic Bézier curve (also known as a conic Bézier curve)
// to the given target passing through the given control point.
func (self *DefaultRasterizer) QuadTo(control, target fixed.Point26_6) {
	cx, cy := self.fixedToFloat32Coords(control)
	tx, ty := self.fixedToFloat32Coords(target)
	self.rasterizer.QuadTo(cx, cy, tx, ty)
}

// Creates a cubic Bézier curve to the given target passing through
// the given control points.
func (self *DefaultRasterizer) CubeTo(controlA, controlB, target fixed.Point26_6) {
	cax, cay := self.fixedToFloat32Coords(controlA)
	cbx, cby := self.fixedToFloat32Coords(controlB)
	tx , ty  := self.fixedToFloat32Coords(target)
	self.rasterizer.CubeTo(cax, cay, cbx, cby, tx, ty)
}

// Satisfies the Rasterizer interface.
func (self *DefaultRasterizer) Rasterize(outline sfnt.Segments, fract fixed.Point26_6) (*image.Alpha, error) {
	// prepare rasterizer
	var size image.Point
	size, self.normOffset, self.rectOffset = figureOutBounds(outline.Bounds(), fract)
	self.rasterizer.Reset(size.X, size.Y)
	self.rasterizer.DrawOp = draw.Src

	// allocate glyph mask
	mask := image.NewAlpha(self.rasterizer.Bounds())

	// process outline
	processOutline(self, outline)

	// since the source texture is a uniform (an image that returns the same
	// color for any coordinate), the value of the point at which we want to
	// start sampling the texture (the fourth parameter) is unimportant.
	self.rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})

	// translate the mask to its final position
	mask.Rect = mask.Rect.Add(self.rectOffset)
	return mask, nil
}

func (self *DefaultRasterizer) fixedToFloat32Coords(point fixed.Point26_6) (float32, float32) {
	x := float32(point.X + self.normOffset.X)/64
	y := float32(point.Y + self.normOffset.Y)/64
	return x, y
}
