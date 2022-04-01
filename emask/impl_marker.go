package emask

import "image"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

// TODO: caches, unsafe access to rasterizer, etc.

type EdgeRastMode int
const (
	EdgeRastFill   EdgeRastMode = 0
	EdgeRastRaw    EdgeRastMode = 1
	EdgeRastHollow EdgeRastMode = 2 // unimplemented, may disappear
)

// An experimental rasterizer using EdgeMarker.
//
// NOTICE: this code still has some issues that make the results not great.
type EdgeMarkerRasterizer struct {
	rasterizer EdgeMarker
	rectOffset image.Point
	normOffset fixed.Point26_6
	cacheSignature uint64
	onChange func(Rasterizer)
	mode EdgeRastMode
}

// Satisfies the UserCfgCacheSignature interface.
func (self *EdgeMarkerRasterizer) SetHighByte(value uint8) {
	self.cacheSignature = uint64(value) << 56
	if self.onChange != nil { self.onChange(self) }
}

// Satisfies the Rasterizer interface.
func (self *EdgeMarkerRasterizer) SetOnChangeFunc(onChange func(Rasterizer)) {
	self.onChange = onChange
}

// Sets the rasterizer mode. Experimental/unsafe.
func (self *EdgeMarkerRasterizer) SetMode(mode EdgeRastMode) {
	self.mode = mode
}

// Get the internal EdgeMarker. Experimental/unsafe.
func (self *EdgeMarkerRasterizer) Marker() *EdgeMarker {
	return &self.rasterizer
}

// Satisfies the Rasterizer interface.
func (self *EdgeMarkerRasterizer) CacheSignature() uint64 {
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

	// rasterize
	switch self.mode {
	case EdgeRastFill:
		index := 0
		for y := 0; y < h; y++ {
			accumulator := float64(0)
			for x := 0; x < w; x++ {
				accumulator += self.rasterizer.Buffer[index]
				mask.Pix[index] = absToUint8(accumulator*255)
				index += 1
			}
		}
	case EdgeRastRaw:
		for i := 0; i < w*h; i++ {
			mask.Pix[i] = absToUint8(self.rasterizer.Buffer[i]*255)
		}
	case EdgeRastHollow:
		panic("EdgeRastHollow unimplemented")
	default:
		panic("unexpected EdgeRastMode")
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
