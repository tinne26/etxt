package mask

import (
	"image"

	"github.com/tinne26/etxt/fract"
	"golang.org/x/image/font/sfnt"
)

var _ Rasterizer = (*EdgeMarkerRasterizer)(nil)

// An alternative to [DefaultRasterizer] that avoids using
// [golang.org/x/image/vector.Rasterizer] under the hood. Results
// are visually very similar, but performance is 3 times worse.
//
// The purpose of this rasterizer is to offer a simpler, more
// readable and [well-documented] version of the algorithm used by
// vector.Rasterizer that anyone can edit, adapt or learn from.
//
// The zero-value will produce jaggy results, as curve segmentation
// parameters are not configured. Reasonable defaults can be set
// through [EdgeMarkerRasterizer.SetCurveThreshold](0.1) and
// [EdgeMarkerRasterizer.SetMaxCurveSplits](8).
//
// [well-documented]: https://github.com/tinne26/etxt/blob/v0.0.9/docs/rasterize-outlines.md
type EdgeMarkerRasterizer struct {
	// All relevant algorithms are implemented inside the unexported
	// edgeMarker type (see mask/edge_marker.go), except for final
	// buffer accumulation which is done directly on the Rasterize()
	// method. The rest is only a wrapper to comply with the
	// emask.Rasterizer interface.
	rasterizer edgeMarker
	onChange   func(Rasterizer)
	rectOffset image.Point
	normOffset fract.Point
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
// Reasonable values range from 0.01 to 1.0. Values outside the [0, 6.5]
// range will be silently clamped. Precision is truncated to three decimal
// places. A good default in my experience is 0.1.
func (self *EdgeMarkerRasterizer) SetCurveThreshold(threshold float64) {
	segmenter := &self.rasterizer.CurveSegmenter
	if self.onChange == nil {
		segmenter.SetThreshold(threshold)
	} else {
		preThreshold := segmenter.curveThresholdThousandths
		segmenter.SetThreshold(threshold)
		if segmenter.curveThresholdThousandths != preThreshold {
			self.onChange(self)
		}
	}
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
// values range from 0 to 10. A good default in my experience is 8, but
// lower values will also work well if you don't need to draw big glyphs.
func (self *EdgeMarkerRasterizer) SetMaxCurveSplits(maxCurveSplits int) {
	segmenter := &self.rasterizer.CurveSegmenter
	if self.onChange == nil {
		segmenter.SetMaxSplits(maxCurveSplits)
	} else {
		preSplits := segmenter.maxCurveSplits
		segmenter.SetMaxSplits(maxCurveSplits)
		if segmenter.maxCurveSplits != preSplits {
			self.onChange(self)
		}
	}
}

// Satisfies the [Rasterizer] interface.
func (self *EdgeMarkerRasterizer) SetOnChangeFunc(onChange func(Rasterizer)) {
	self.onChange = onChange
}

// Satisfies the [Rasterizer] interface. The signature for the
// edge marker rasterizer has the following shape:
//   - 0xFF00000000000000 unused bits customizable through type embedding.
//   - 0x00FF000000000000 bits being 0xE6 (self signature byte).
//   - 0x0000FFFFFF000000 bits being zero, currently undefined.
//   - 0x0000000000FFFFFF bits representing the curve segmenter configuration.
func (self *EdgeMarkerRasterizer) Signature() uint64 {
	segmenter := &self.rasterizer.CurveSegmenter
	return 0x00E6000000000000 | segmenter.Signature()
}

// See [DefaultRasterizer.MoveTo]().
func (self *EdgeMarkerRasterizer) MoveTo(point fract.Point) {
	x, y := point.AddPoint(self.normOffset).ToFloat64s()
	self.rasterizer.MoveTo(x, y)
}

// See [DefaultRasterizer.LineTo]().
func (self *EdgeMarkerRasterizer) LineTo(point fract.Point) {
	x, y := point.AddPoint(self.normOffset).ToFloat64s()
	self.rasterizer.LineTo(x, y)
}

// See [DefaultRasterizer.QuadTo]().
func (self *EdgeMarkerRasterizer) QuadTo(control, target fract.Point) {
	cx, cy := control.AddPoint(self.normOffset).ToFloat64s()
	tx, ty := target.AddPoint(self.normOffset).ToFloat64s()
	self.rasterizer.QuadTo(cx, cy, tx, ty)
}

// See [DefaultRasterizer.CubeTo]().
func (self *EdgeMarkerRasterizer) CubeTo(controlA, controlB, target fract.Point) {
	cax, cay := controlA.AddPoint(self.normOffset).ToFloat64s()
	cbx, cby := controlB.AddPoint(self.normOffset).ToFloat64s()
	tx, ty := target.AddPoint(self.normOffset).ToFloat64s()
	self.rasterizer.CubeTo(cax, cay, cbx, cby, tx, ty)
}

// Satisfies the [Rasterizer] interface.
func (self *EdgeMarkerRasterizer) Rasterize(outline sfnt.Segments, origin fract.Point) (*image.Alpha, error) {
	// get outline bounds
	fbounds := outline.Bounds()
	bounds := fract.Rect{
		Min: fract.UnitsToPoint(fract.Unit(fbounds.Min.X), fract.Unit(fbounds.Min.Y)),
		Max: fract.UnitsToPoint(fract.Unit(fbounds.Max.X), fract.Unit(fbounds.Max.Y)),
	}

	// prepare rasterizer
	var width, height int
	width, height, self.normOffset, self.rectOffset = figureOutBounds(bounds, origin)
	buffer := &self.rasterizer.Buffer
	buffer.Resize(width, height)

	// process outline
	processOutline(self, outline)

	// allocate glyph mask and apply buffer accumulation
	// (this takes around 50% of the time of the process)
	mask := image.NewAlpha(image.Rect(0, 0, buffer.Width, buffer.Height))
	buffer.AccumulateUint8(mask.Pix)

	// translate the mask to its final position
	mask.Rect = mask.Rect.Add(self.rectOffset)
	return mask, nil
}
