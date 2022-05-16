package emask

import "math"
import "image"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

// An alternative to vector.Rasterizer. Results are visually
// very similar, but performance is 3 times worse.
//
// The purpose of this rasterizer is to offer a simpler, more
// readable and [well-documented] version of the algorithm used by
// vector.Rasterizer that anyone can edit, adapt or learn from.
//
// The zero-value is usable but will produce jaggy results, as curve
// segmentation parameters are not configured. Use NewStdEdgeMarkerRasterizer()
// if you prefer a pre-configured rasterizer. You may also configure the
// rasterizer manually through SetCurveThreshold() and SetMaxCurveSplits().
//
// [well-documented]: https://github.com/tinne26/etxt/blob/main/docs/rasterize-outlines.md
type EdgeMarkerRasterizer struct {
	// All relevant algorithms are implemented inside the unexported
	// edgeMarker type (see emask/edge_marker.go), except for final
	// buffer accumulation which is done directly on the Rasterize()
	// method. The rest is only a wrapper to comply with the
	// emask.Rasterizer interface.
	rasterizer edgeMarker
	onChange func(Rasterizer)
	cacheSignature uint64
	rectOffset image.Point
	normOffset fixed.Point26_6
}

func NewStdEdgeMarkerRasterizer() *EdgeMarkerRasterizer {
	rast := &EdgeMarkerRasterizer{}
	rast.SetCurveThreshold(0.2)
	rast.SetMaxCurveSplits(8) // this is excessive for most glyph rendering
	return rast
}

// Satisfies the UserCfgCacheSignature interface.
func (self *EdgeMarkerRasterizer) SetHighByte(value uint8) {
	self.cacheSignature = uint64(value) << 56
	if self.onChange != nil { self.onChange(self) }
}

// Sets the threshold distance to use when splitting Bézier curves into
// linear segments. If a linear segment misses the curve by more than
// the threshold value, the curve will be split. Otherwise, the linear
// segment will be used to approximate it.
//
// Values very close to zero could prevent the algorithm from converging
// due to floating point instability, but the MaxCurveSplits cutoff will
// prevent infinite looping anyway.
//
// Reasonable values range from 0.01 to 1.0. NewStdEdgeMarkerRasterizer()
// uses 0.2 by default.
func (self *EdgeMarkerRasterizer) SetCurveThreshold(threshold float32) {
	self.rasterizer.SetCurveThreshold(threshold)
	bits := math.Float32bits(threshold)
	self.cacheSignature &= 0xFFFFFFFF00000000
	self.cacheSignature |= uint64(bits)
	if self.onChange != nil { self.onChange(self) }
}

// Sets the maximum amount of times a curve can be recursively split
// into subsegments while trying to approximate it.
//
// The maximum number of segments that will approximate a curve is
// 2^maxCurveSplits.
//
// This value is typically used as a cutoff to prevent low curve thresholds
// from making the curve splitting process too slow, but it can also be used
// creatively to get jaggy results instead of smooth curves.
//
// Values outside the [0, 255] range will be silently clamped. Reasonable
// values range from 0 to 10. NewStdEdgeMarkerRasterizer() uses 8 by default.
func (self *EdgeMarkerRasterizer) SetMaxCurveSplits(maxCurveSplits int) {
	self.rasterizer.SetMaxCurveSplits(maxCurveSplits)
	self.cacheSignature &= 0xFFFFFF00FFFFFFFF
	self.cacheSignature |= uint64(self.rasterizer.maxCurveSplits) << 32
	if self.onChange != nil { self.onChange(self) }
}

// Satisfies the Rasterizer interface.
func (self *EdgeMarkerRasterizer) SetOnChangeFunc(onChange func(Rasterizer)) {
	self.onChange = onChange
}

// Satisfies the Rasterizer interface.
func (self *EdgeMarkerRasterizer) CacheSignature() uint64 {
	self.cacheSignature &= 0xFF00FFFFFFFFFFFF
	self.cacheSignature |= 0x00E6000000000000
	return self.cacheSignature
}

// Satisfies the vectorTracer interface.
func (self *EdgeMarkerRasterizer) MoveTo(point fixed.Point26_6) {
	x, y := self.fixedToFloat64Coords(point)
	self.rasterizer.MoveTo(x, y)
}

// Satisfies the vectorTracer interface.
func (self *EdgeMarkerRasterizer) LineTo(point fixed.Point26_6) {
	x, y := self.fixedToFloat64Coords(point)
	self.rasterizer.LineTo(x, y)
}

// Satisfies the vectorTracer interface.
func (self *EdgeMarkerRasterizer) QuadTo(control, target fixed.Point26_6) {
	cx, cy := self.fixedToFloat64Coords(control)
	tx, ty := self.fixedToFloat64Coords(target)
	self.rasterizer.QuadTo(cx, cy, tx, ty)
}

// Satisfies the vectorTracer interface.
func (self *EdgeMarkerRasterizer) CubeTo(controlA, controlB, target fixed.Point26_6) {
	cax, cay := self.fixedToFloat64Coords(controlA)
	cbx, cby := self.fixedToFloat64Coords(controlB)
	tx , ty  := self.fixedToFloat64Coords(target)
	self.rasterizer.CubeTo(cax, cay, cbx, cby, tx, ty)
}

// Satisfies the Rasterizer interface.
func (self *EdgeMarkerRasterizer) Rasterize(outline sfnt.Segments, fract fixed.Point26_6) (*image.Alpha, error) {
	// prepare rasterizer
	var size image.Point
	size, self.normOffset, self.rectOffset = figureOutBounds(outline.Bounds(), fract)
	self.rasterizer.Resize(size.X, size.Y)

	// allocate glyph mask
	w, h := self.rasterizer.Width, self.rasterizer.Height
	mask := image.NewAlpha(image.Rect(0, 0, w, h))

	// process outline
	processOutline(self, outline)

	// rasterize applying buffer accumulation
	// (this takes around 40% of the time of the process)
	index := 0
	for y := 0; y < h; y++ {
		accumulator := float64(0)
		for x := 0; x < w; x++ {
			accumulator += self.rasterizer.Buffer[index]
			mask.Pix[index] = absToUint8(accumulator*255)
			index += 1
		}
	}

	// translate the mask to its final position
	mask.Rect = mask.Rect.Add(self.rectOffset)
	return mask, nil
}

func (self *EdgeMarkerRasterizer) fixedToFloat64Coords(point fixed.Point26_6) (float64, float64) {
	x := float64(point.X + self.normOffset.X)/64
	y := float64(point.Y + self.normOffset.Y)/64
	return x, y
}

func absToUint8(acc float64) uint8 {
	if acc < 0 { acc = -acc }
	if acc > 255 { acc = 255 }
	return uint8(acc)
}
