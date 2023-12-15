package mask

import "image"
import "image/draw"

import "golang.org/x/image/vector"
import "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/fract"

var _ Rasterizer = (*DefaultRasterizer)(nil)

// The DefaultRasterizer is a wrapper to make [golang.org/x/image/vector.Rasterizer]
// conform to the [Rasterizer] interface.
type DefaultRasterizer struct {
	rasterizer vector.Rasterizer
	normOffset fract.Point // offset to normalize points to the positive
	                       // quadrant starting from the fractional coords
	onChange func(Rasterizer)

	// Notice that the x/image/vector rasterizer expects coords in the
	// positive quadrant, which is why we need so many offsets here.
}

// Satisfies the [Rasterizer] interface.
func (self *DefaultRasterizer) SetOnChangeFunc(onChange func(Rasterizer)) {
	self.onChange = onChange
}

// Satisfies the [Rasterizer] interface. The signature for the
// default rasterizer is always zero, but may be customized as
// you want through type embedding and method overriding.
func (self *DefaultRasterizer) Signature() uint64 { return 0 }

// Moves the current position to the given point.
func (self *DefaultRasterizer) MoveTo(point fract.Point) {
	x, y := point.AddPoint(self.normOffset).ToFloat32s()
	self.rasterizer.MoveTo(x, y)
}

// Creates a straight boundary from the current position to the given point.
func (self *DefaultRasterizer) LineTo(point fract.Point) {
	x, y := point.AddPoint(self.normOffset).ToFloat32s()
	self.rasterizer.LineTo(x, y)
}

// Creates a quadratic Bézier curve (also known as a conic Bézier curve)
// to the given target passing through the given control point.
func (self *DefaultRasterizer) QuadTo(control, target fract.Point) {
	cx, cy := control.AddPoint(self.normOffset).ToFloat32s()
	tx, ty := target.AddPoint(self.normOffset).ToFloat32s()
	self.rasterizer.QuadTo(cx, cy, tx, ty)
}

// Creates a cubic Bézier curve to the given target passing through
// the given control points.
func (self *DefaultRasterizer) CubeTo(controlA, controlB, target fract.Point) {
	cax, cay := controlA.AddPoint(self.normOffset).ToFloat32s()
	cbx, cby := controlB.AddPoint(self.normOffset).ToFloat32s()
	tx , ty  := target.AddPoint(self.normOffset).ToFloat32s()
	self.rasterizer.CubeTo(cax, cay, cbx, cby, tx, ty)
}

// Satisfies the [Rasterizer] interface.
func (self *DefaultRasterizer) Rasterize(outline sfnt.Segments, origin fract.Point) (*image.Alpha, error) {
	// get outline bounds
	fbounds := outline.Bounds()
	bounds := fract.Rect{
		Min: fract.UnitsToPoint(fract.Unit(fbounds.Min.X), fract.Unit(fbounds.Min.Y)),
		Max: fract.UnitsToPoint(fract.Unit(fbounds.Max.X), fract.Unit(fbounds.Max.Y)),
	}

	// prepare rasterizer
	var width, height int
	var rectOffset image.Point
	width, height, self.normOffset, rectOffset = figureOutBounds(bounds, origin)
	self.rasterizer.Reset(width, height)
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
	mask.Rect = mask.Rect.Add(rectOffset)
	return mask, nil
}
