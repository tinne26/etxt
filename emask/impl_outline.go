package emask

import "image"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/efixed"

// TODO: extension of joints must be limited through special mechanisms,
//       otherwise they can go wild on some aberrant cases

// NOTICE: work in progress. There are a couple big clipping issues
// that will easily cause panics, and most likely still a few minor
// bugs on polygon filling edge cases. Clipping is tricky due to Bézier
// cusps and similar situations where two lines end up joining at
// very tight angles, but also in some edge cases where natural path
// intersections will go outside the intended paths area due to thickness
// becoming larger than path length and multiple paths colliding and stuff.
//
// The algorithm is also quite slow and there's much room for improvement,
// but I'm still focusing on the baseline implementation.
//
// In web API terms, the line cap is "butt" and the line joint is "miter".
type OutlineRasterizer struct {
	rasterizer outliner
	onChange func(Rasterizer)
	cacheSignature uint64
	rectOffset image.Point
	normOffset fixed.Point26_6
}

func NewOutlineRasterizer(outlineThickness float64) *OutlineRasterizer {
	rast := &OutlineRasterizer{}
	rast.SetThickness(outlineThickness)
	rast.rasterizer.CurveSegmenter.SetThreshold(1/1024)
	rast.rasterizer.CurveSegmenter.SetMaxSplits(8) // TODO: store somewhere
	rast.SetMarginFactor(2.0)
	return rast
}

// Satisfies the [UserCfgCacheSignature] interface.
func (self *OutlineRasterizer) SetHighByte(value uint8) {
	self.cacheSignature = uint64(value) << 56
	if self.onChange != nil { self.onChange(self) }
}

// Sets the outline thickness. Values must be in the [0.1, 1024] range.
func (self *OutlineRasterizer) SetThickness(thickness float64) {
	thickness = self.rasterizer.SetThickness(thickness)
	self.cacheSignature &= 0xFFFFFFFFFFF00000
	self.cacheSignature |= uint64(thickness*1024) - 1
	if self.onChange != nil { self.onChange(self) }
}

// When two lines of the outline connect at a tight angle, the resulting
// vertex may extend far beyond 'thickness' distance. The margin factor
// allows setting a limit, in multiples of 'thickness', to adjust the
// paths so they don't extend further away than intended.
//
// The default value is 2. Valid values range from 1 to 16.
func (self *OutlineRasterizer) SetMarginFactor(factor float64) {
	// TODO: document how multiples provide coverage up to different
	// angles. TODO: this is still causing panics.
	self.rasterizer.SetMarginFactor(factor)
}

// Satisfies the [Rasterizer] interface.
func (self *OutlineRasterizer) SetOnChangeFunc(onChange func(Rasterizer)) {
	self.onChange = onChange
}

// Satisfies the [Rasterizer] interface.
func (self *OutlineRasterizer) CacheSignature() uint64 {
	self.cacheSignature &= 0xFF00FFFFFFFFFFFF
	self.cacheSignature |= 0x0037000000000000
	return self.cacheSignature
}

// Satisfies the unexported vectorTracer interface.
func (self *OutlineRasterizer) MoveTo(point fixed.Point26_6) {
	x, y := self.fixedToFloat64Coords(point)
	self.rasterizer.MoveTo(x, y)
}

// Satisfies the unexported vectorTracer interface.
func (self *OutlineRasterizer) LineTo(point fixed.Point26_6) {
	x, y := self.fixedToFloat64Coords(point)
	self.rasterizer.LineTo(x, y)
}

// Satisfies the unexported vectorTracer interface.
func (self *OutlineRasterizer) QuadTo(control, target fixed.Point26_6) {
	cx, cy := self.fixedToFloat64Coords(control)
	tx, ty := self.fixedToFloat64Coords(target)
	self.rasterizer.QuadTo(cx, cy, tx, ty)
}

// Satisfies the unexported vectorTracer interface.
func (self *OutlineRasterizer) CubeTo(controlA, controlB, target fixed.Point26_6) {
	cax, cay := self.fixedToFloat64Coords(controlA)
	cbx, cby := self.fixedToFloat64Coords(controlB)
	tx , ty  := self.fixedToFloat64Coords(target)
	self.rasterizer.CubeTo(cax, cay, cbx, cby, tx, ty)
}

// Satisfies the Rasterizer interface.
func (self *OutlineRasterizer) Rasterize(outline sfnt.Segments, fract fixed.Point26_6) (*image.Alpha, error) {
	// prepare rasterizer
	var size image.Point
	bounds := outline.Bounds()
	margin := efixed.FromFloat64RoundAwayZero(self.rasterizer.MaxMargin())
	bounds.Min.X -= margin
	bounds.Max.X += margin
	bounds.Min.Y -= margin
	bounds.Max.Y += margin
	size, self.normOffset, self.rectOffset = figureOutBounds(bounds, fract)
	buffer := &self.rasterizer.Buffer
	buffer.Resize(size.X, size.Y) // also clears the buffer
	// TODO: use a buffer8 uint8 buffer and set manually
	//       .Width, .Height and swap the .Buffer directly

	// process outline
	processOutline(self, outline)

	// allocate glyph mask and move results from buffer
	mask := image.NewAlpha(image.Rect(0, 0, buffer.Width, buffer.Height))
	for i := 0; i < len(buffer.Values); i++ {
		mask.Pix[i] = uint8(clampUnit64(buffer.Values[i])*255)
	}

	// translate the mask to its final position
	mask.Rect = mask.Rect.Add(self.rectOffset)
	return mask, nil
}

func (self *OutlineRasterizer) fixedToFloat64Coords(point fixed.Point26_6) (float64, float64) {
	x := float64(point.X + self.normOffset.X)/64
	y := float64(point.Y + self.normOffset.Y)/64
	return x, y
}
