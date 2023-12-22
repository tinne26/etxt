package mask

import "math/bits"
import "image"
import "image/draw"

import "golang.org/x/image/vector"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

var _ Rasterizer = (*FauxRasterizer)(nil)

// TODO: there's an index out of bounds error when we set
//       high values for SetExtraWidth, take a look at that

// A rasterizer to draw oblique and faux-bold text. For high quality
// results, please use the font's italic and bold versions directly
// instead of these fake effects.
//
// In general, the performance of [FauxRasterizer] without effects is very
// similar to [DefaultRasterizer]. Using reasonable skew factors for
// oblique text tends to increase the rasterization time around 15%, and
// using faux-bold increases the rasterization time in 60%, but it depends
// a lot on how extreme the effects are.
//
// This rasterizer was created mostly to serve as an example of how to
// create modified rasterizers, featuring both modification of glyph
// control points (oblique) and post-processing of the generated mask
// (faux-bold).
type FauxRasterizer struct {
	// similar fields to DefaultRasterizer
	rasterizer vector.Rasterizer
	onChange func(Rasterizer)
	auxOnChange func(*FauxRasterizer)
	signature uint64
	normOffset fract.Point

	// skewing-related fields (oblique)
	skewing float32 // between -1 (45 degrees) and 1 (-45 degrees)
	                // (quantized to be representable without loss
					    //  in 16 bits)

	// extra width (faux-bold) related fields
	xwidth fract.Unit
	xwidthTailMod uint16
	xwidthTail []uint8 // internal implementation detail
}

// Sets the oblique skewing factor, which is expected to be in [-1, 1].
// Values outside this range will be silently clamped.
//
// This factor is internally defined to represent skews ranging from 45 to -45
// degrees. Some practical examples:
//  - A skew factor of -1 will rasterize glyphs tilted 45 degrees left (backwards leaning).
//  - A skew factor of 1 will rasterize glyphs tilted 45 degrees right (forwards leaning).
//  - A skew factor of 0 will rasterize glyphs without any tilt.
//  - A skew factor of 0.5 will rasterize glyphs tilted 22.5 degrees right (forwards leaning).
// Most italic fonts have an angle between 6 and 9 degrees, which correspond
// to skew factors in the [0.13, 0.2] range.
func (self *FauxRasterizer) SetSkewFactor(factor float32) {
	// normalize and store new skewing factor
	if factor == 0 {
		if self.skewing == 0 { return }
		self.skewing = 0
		self.signature = self.signature & 0xFFFF0FFF0000FFFF
	} else {
		if factor >  1.0 { factor =  1.0 }
		if factor < -1.0 { factor = -1.0 }
		skewUint16  := uint16FromUnitFP32(factor)
		skewMask    := uint64(skewUint16) << 16
		if (self.signature & 0x00000000_FFFF0000) == skewMask && (self.signature & 0x00001000_00000000) != 0 {
			return // early return
		}
		skewFloat32 := unitFP32FromUint16(skewUint16)
		self.skewing = skewFloat32

		// update signature
		self.signature = self.signature & 0xFFFF0FFF_0000FFFF
		self.signature |= 0x0000100000000000 // flag for "active italics"
		self.signature |= skewMask
	}

	self.notifyChange()
}

// Gets the skewing factor [-1.0, 1.0] used for the oblique style.
// See [FauxRasterizer.SetSkewFactor]() if you need details on how
// to interpret skew values.
func (self *FauxRasterizer) GetSkewFactor() float32 {
	return self.skewing
}

// Sets the extra width for the faux-bold. Values outside the [0, 1024]
// range will be clamped. Fractional values are allowed, but internally
// the decimal part will be quantized to 1/64ths of a pixel.
//
// Important: when extra width is used for faux-bold, the glyphs will
// become wider. If you want to adapt the positioning of the glyphs to
// account for this widening, you can use a [sizer.PaddedAdvanceSizer],
// link the rasterizer to it through [FauxRasterizer.SetAuxOnChangeFunc]()
// and update the padding with the value of [FauxRasterizer.GetExtraWidth](),
// for example.
//
// [sizer.PaddedAdvanceSizer]: https://pkg.go.dev/github.com/tinne26/etxt/sizer@v0.0.9-alpha.7#PaddedAdvanceSizer
func (self *FauxRasterizer) SetExtraWidth(extraWidth float32) {
	// normalize and store new skewing factor
	if extraWidth <= 0 {
		if self.xwidth == 0 { return } // shortcut
		self.xwidth = 0
		self.signature = self.signature & 0xFFFFF0FF_FFFF0000
	} else {
		if extraWidth > 1024.0 { extraWidth = 1024 }
		fractExtraWidth := fract.FromFloat64Down(float64(extraWidth))
		if fractExtraWidth == 0 { fractExtraWidth = 1 } // prevent rounding to zero
		if self.xwidth == fractExtraWidth { return } // early return
		self.xwidth = fractExtraWidth
		xwidthWhole := self.xwidth.ToIntFloor()
		if len(self.xwidthTail) < xwidthWhole {
			if self.xwidthTail == nil {
				self.xwidthTail = make([]uint8, 8)
				self.xwidthTailMod = 7
			} else {
				targetSize := uint16RoundToNextPow2(uint16(xwidthWhole))
				if targetSize == 1 { panic("unreachable") }
				self.xwidthTailMod = targetSize - 1
				if uint16(cap(self.xwidthTail)) >= targetSize {
					self.xwidthTail = self.xwidthTail[0 : targetSize]
				} else {
					self.xwidthTail = make([]uint8, targetSize)
				}
			}
		}

		// update cache signature
		self.signature = self.signature & 0xFFFFF0FF_FFFF0000
		self.signature |= 0x00000B00_00000000 // flag for "active bold"
		self.signature |= uint64(self.xwidth)
	}

	self.notifyChange()
}

// round the given uint16 to the next power of two (stays
// as it is if the value is already a power of two)
func uint16RoundToNextPow2(value uint16) uint16 {
	if value == 1 { return 2 }
	if bits.OnesCount16(value) <= 1 { return value } // (already a pow2)
	return uint16(1) << (16 - bits.LeadingZeros16(value))
}

// Gets the extra width (in pixels, possibly fractional)
// used for the faux-bold style.
func (self *FauxRasterizer) GetExtraWidth() float32 {
	return self.xwidth.ToFloat32()
}

// Satisfies the [Rasterizer] interface. The signature for the
// faux rasterizer has the following shape:
//  - 0xFF00000000000000 unused bits customizable through type embedding.
//  - 0x00FF000000000000 bits being 0xFA (self signature byte).
//  - 0x0000F00000000000 bits being 0x1 if italics are enabled.
//  - 0x00000F0000000000 bits being 0xB if bold is enabled.
//  - 0x000000FF00000000 bits being zero, currently undefined.
//  - 0x00000000FFFF0000 bits encoding the skew [-1, 1] in the [0, 65535]
//    range, with the "zero skew" not having a representation (signatures
//    are still different due to the "italics-enabled" flag).
//  - 0x000000000000FFFF bits encoding the extra bold width in 64ths of
//    a pixel and encoded in a uint16.
func (self *FauxRasterizer) Signature() uint64 {
	return 0x00FA0000_00000000 | self.signature
}

func (self *FauxRasterizer) pointToFloat32Coords(point fract.Point) (float32, float32) {
	// careful with the operation order here; the skewing needs
	// to be applied before the Y is adjusted with the normOffset
	x := (point.X + self.normOffset.X).ToFloat32() - point.Y.ToFloat32()*self.skewing
	y := (point.Y + self.normOffset.Y).ToFloat32()
	return x, y
}

// Satisfies the [Rasterizer] interface.
func (self *FauxRasterizer) Rasterize(outline sfnt.Segments, origin fract.Point) (*image.Alpha, error) {
	rectOffset := self.prepareForOutline(outline, origin)
	mask := image.NewAlpha(self.rasterizer.Bounds())
	processOutline(self, outline)
	self.rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	mask.Rect = mask.Rect.Add(rectOffset)

	if self.xwidth > 0 { // faux bold post-processing
		self.applyExtraWidth(mask.Pix, mask.Stride)
	}
	return mask, nil
}

// Like [FauxRasterizer.SetOnChangeFunc], but not reserved for internal
// [Renderer] use. This is provided so you can link a custom [sizer.Sizer] to
// the rasterizer and get notified when its configuration changes.
//
// [Renderer]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.7#Renderer
// [sizer.Sizer]: https://pkg.go.dev/github.com/tinne26/etxt/sizer@v0.0.9-alpha.7#Sizer
func (self *FauxRasterizer) SetAuxOnChangeFunc(onChange func(*FauxRasterizer)) {
	self.auxOnChange = onChange
}

func (self *FauxRasterizer) notifyChange() {
	if self.onChange    != nil { self.onChange(self)    }
	if self.auxOnChange != nil { self.auxOnChange(self) }
}

func (self *FauxRasterizer) prepareForOutline(outline sfnt.Segments, origin fract.Point) image.Point {
	// get outline bounds
	fbounds := outline.Bounds()
	bounds := fract.Rect{
		Min: fract.UnitsToPoint(fract.Unit(fbounds.Min.X), fract.Unit(fbounds.Min.Y)),
		Max: fract.UnitsToPoint(fract.Unit(fbounds.Max.X), fract.Unit(fbounds.Max.Y)),
	}

	// adjust the bounds accounting for skewing
	if self.skewing != 0 {
		// to understand this mysterious code you have to unroll it, know that 
		// ascendents are negative, check all the tricky sign combinations
		// between bounds and skewings and see that it resolves to this
		shiftMin := fract.FromFloat64(bounds.Min.Y.ToFloat64()*float64(self.skewing))
		shiftMax := fract.FromFloat64(bounds.Max.Y.ToFloat64()*float64(self.skewing))
		if self.skewing >= 0 { shiftMin, shiftMax = shiftMax, shiftMin }
		bounds.Min.X -= shiftMin
		bounds.Max.X -= shiftMax
	}

	// adjust the bounds accounting for faux-bold extra width
	if self.xwidth > 0 {
		bounds.Max.X += self.xwidth.Ceil()
	}

	// similar to default rasterizer
	var width, height int
	var rectOffset image.Point
	width, height, self.normOffset, rectOffset = figureOutBounds(bounds, origin)
	self.rasterizer.Reset(width, height)
	self.rasterizer.DrawOp = draw.Src
	return rectOffset
}

// ==== EXTRA WIDTH COMPUTATIONS ====
// I got very traumatized trying to figure out all this fake-bold stuff.
// Just use a proper bold font and leave me alone.
//
// ...
//
// Better faux-bold would have to be done through shape expansion anyway,
// working directly with the outline points, but that's tricky to do (e.g:
// github.com/libass/libass/blob/7bf4bee0fc9a1d6257a105a3c19df6cf08733f8e/
// libass/ass_outline.c#L499)... but even freetype's faux-bold is not perfect.

func (self *FauxRasterizer) applyExtraWidth(pixels []uint8, stride int) {
	// extra width is applied independently to each row
	for x := 0; x < len(pixels); x += stride {
		self.applyRowExtraWidth(pixels[x : x + stride], pixels, x, stride)
	}
}

func (self *FauxRasterizer) applyRowExtraWidth(row, pixels []uint8, start, stride int) {
	var peakAlpha uint8
	var twoPixSwap bool // flag for "two-pixel-stem" fix

	// for each row, the idea is to ascend to the biggest alpha
	// values first, and then when falling apply the extra width,
	// mostly as a keep-max-of-last-n-alpha-values.
	for index := 0; index < len(row); {
		index, peakAlpha = self.extraWidthRowAscend(row, index)
		if peakAlpha == 0 { return }
		peakAlpha, twoPixSwap = self.peakAlphaFix(row, pixels, index, start, stride, peakAlpha)
		index = self.extraWidthRowFall(row, index, peakAlpha, twoPixSwap)
	}
}

func (self *FauxRasterizer) peakAlphaFix(row, pixels []uint8, index, start, stride int, peakAlpha uint8) (uint8, bool) {
	if peakAlpha == 255 || self.xwidth.Floor() == 0 { return peakAlpha, false }

	// check boundaries
	if index < 2 { return peakAlpha, false }
	if index + 1 >= len(row) { return peakAlpha, false }
	aboveIndex := (start + index - 1 - stride)
	belowIndex := (start + index - 1 + stride)
	if aboveIndex < 0 || belowIndex > len(pixels) { return peakAlpha, false }

	// "in stem" heuristic
 	pixAbove := (pixels[aboveIndex - 1] > 0 || pixels[aboveIndex] > 0 || pixels[aboveIndex + 1] > 0)
	pixBelow := (pixels[belowIndex - 1] > 0 || pixels[belowIndex] > 0 || pixels[belowIndex + 1] > 0)
	if !pixAbove || !pixBelow { return peakAlpha, false }

	// handle the edge case of two-pixel stem
	if index >= 3 && row[index] == 0 && row[index - 2] != 0 && row[index - 3] == 0 {
		return 255, true // two-pix-stem swap is necessary!
	}

	return 255, false
}

// Returns the first index after the alpha peak, along with the peak value.
func (self *FauxRasterizer) extraWidthRowAscend(row []uint8, index int) (int, uint8) {
	peakAlpha := uint8(0)
	prevAlpha := uint8(0)
	for ; index < len(row); index++ {
		currAlpha := row[index]
		if currAlpha == prevAlpha { continue }
		if currAlpha  < prevAlpha { return index, peakAlpha }
		if currAlpha  > peakAlpha { peakAlpha = currAlpha }
		prevAlpha = currAlpha
	}
	return 0, 0
}

// The tricky part. As mentioned before, the main idea is to
// keep-max-of-last-n-alpha-values, but... *trauma intensifies*
func (self *FauxRasterizer) extraWidthRowFall(row []uint8, index int, peakAlpha uint8, twoPixSwap bool) int {
	// apply the whole width part...
	whole := uint16(self.xwidth >> 6) // fixed point arithmetic optimization
	if whole == 0 { // ...unless there's no whole part, I guess
		return self.extraWidthRowFractFall(row, index, peakAlpha)
	}

	peakAlphaIndex := index - 1
	realPeakAlpha  := row[peakAlphaIndex]
	for n := uint16(0); n < whole; n++ {
		currAlpha := row[index]
		if currAlpha >= peakAlpha {
			row[peakAlphaIndex] = peakAlpha
			return index
		}
		row[index] = peakAlpha
		self.xwidthTail[n] = currAlpha
		index += 1
	}

	// two-pixel-stem swap correction
	if twoPixSwap {
		row[peakAlphaIndex] = peakAlpha
		row[index - 1] = realPeakAlpha
		self.xwidthTail[0] = realPeakAlpha
		peakAlpha = realPeakAlpha
	}

	// we are done with the whole width peak part. now... what's this?
	mod := self.xwidthTailMod
	if whole > 1 { self.backfixTail(whole) }

	// prepare variables to propagate the tail
	tailIndex   := uint16(0)
	prevAlpha   := peakAlpha
	prevTailAdd := peakAlpha
	if twoPixSwap { tailIndex = 1 }

	// propagate the tail
	for index < len(row) {
		tailAlpha := self.xwidthTail[tailIndex]
		newAlpha  := self.interpolateForExtraWidth(prevAlpha, tailAlpha)
		currAlpha := row[index]
		if currAlpha >= newAlpha { return index } // not falling anymore
		row[index] = newAlpha

		// put current alpha on the tail
		newTailIndex := (tailIndex + whole) & mod
		self.xwidthTail[newTailIndex] = currAlpha
		if currAlpha > prevTailAdd { // tests recommended me this.
			self.backfixTailGen(whole, newTailIndex, mod)
		}
		prevTailAdd = currAlpha
		tailIndex = (tailIndex + 1) & mod

		// please let's go to the next value already
		prevAlpha = tailAlpha
		index += 1
	}
	return index
}

// while we were filling a row with peak values, maybe the values
// that we were overwritting had some ups and downs, and while in
// other parts of the code we can control for that manually, in
// this part they might have gone unnoticed. this function corrects
// this and normalizes the tail to their max possible values.
// expects the tail to start at index = 0.
func (self *FauxRasterizer) backfixTail(whole uint16) {
	// this code is so ugly
	i := whole - 1
	max := self.xwidthTail[i]
	i -= 1
	for {
		value := self.xwidthTail[i]
		if value > max {
			max = value
		} else if value < max {
			self.xwidthTail[i] = max
		}
		if i == 0 { return }
		i -= 1
	}
}

// like backfixTail, but without starting at 0
func (self *FauxRasterizer) backfixTailGen(whole uint16, lastIndex uint16, mod uint16) {
	if whole <= 1 { return }
	max := self.xwidthTail[lastIndex]
	whole -= 1
	if lastIndex > 0 { lastIndex -= 1 } else { lastIndex = mod }
	for {
		value := self.xwidthTail[lastIndex]
		if value > max {
			max = value
		} else if value < max {
			self.xwidthTail[lastIndex] = max
		}
		whole -= 1
		if whole == 0 { return }
		if lastIndex > 0 { lastIndex -= 1 } else { lastIndex = mod } // can you hear gofmt scream already?
	}
}

// like extraWidthRowFall, but when the whole part of the extra
// width is zero and there's only a fractional part to add
func (self *FauxRasterizer) extraWidthRowFractFall(row []uint8, index int, peakAlpha uint8) int {
	prevAlpha := peakAlpha
	for ; index < len(row); index++ {
		currAlpha := row[index]
		newAlpha  := self.interpolateForExtraWidth(prevAlpha, currAlpha)
		if currAlpha >= newAlpha { return index }
		row[index] = newAlpha
		prevAlpha  = currAlpha
	}
	return index
}

func (self *FauxRasterizer) interpolateForExtraWidth(prevAlpha, currAlpha uint8) uint8 {
	// some fixed point optimized arithmetic
	fractPart := self.xwidth.FractShift()
	prevWeight := uint8((fractPart*fract.Unit(prevAlpha) + 32) >> 6)
	currWeight := uint8(((64 - fractPart)*fract.Unit(currAlpha) + 32) >> 6)
	return prevWeight + currWeight
}

// ==== FROM HERE ON METHODS ARE THE SAME AS DEFAULT RASTERIZER ====
// (...so you can ignore them, nothing new here)

// See [DefaultRasterizer.MoveTo]().
func (self *FauxRasterizer) MoveTo(point fract.Point) {
	x, y := self.pointToFloat32Coords(point)
	self.rasterizer.MoveTo(x, y)
}

// See [DefaultRasterizer.LineTo]().
func (self *FauxRasterizer) LineTo(point fract.Point) {
	x, y := self.pointToFloat32Coords(point)
	self.rasterizer.LineTo(x, y)
}

// See [DefaultRasterizer.QuadTo]().
func (self *FauxRasterizer) QuadTo(control, target fract.Point) {
	cx, cy := self.pointToFloat32Coords(control)
	tx, ty := self.pointToFloat32Coords(target)
	self.rasterizer.QuadTo(cx, cy, tx, ty)
}

// See [DefaultRasterizer.CubeTo]().
func (self *FauxRasterizer) CubeTo(controlA, controlB, target fract.Point) {
	cax, cay := self.pointToFloat32Coords(controlA)
	cbx, cby := self.pointToFloat32Coords(controlB)
	tx , ty  := self.pointToFloat32Coords(target)
	self.rasterizer.CubeTo(cax, cay, cbx, cby, tx, ty)
}

// Satisfies the [Rasterizer] interface.
func (self *FauxRasterizer) SetOnChangeFunc(onChange func(Rasterizer)) {
	self.onChange = onChange
}

// Helper for storing a float32 in [-1, 1] range as uint16.
func uint16FromUnitFP32(n float32) uint16 {
	if n >= 0 {
		value := uint16(n*32768) + 32767
		if value < 32768 { return 32768 }
		return value
	} else {
		value := 32768 - uint16(n*-32768)
		if value >= 32768 { return 32767 }
		return value
	}
}

// Helper for converting a uint16 to a float32 in [-1, 1] range.
func unitFP32FromUint16(n uint16) float32 {
	if n >= 32768 { // [32768, 65535] => [0.XXX, 1.0]
		return float32(n - 32767)/float32(32768)
	} else { // [0, 32767] => [-1, -0.XXX]
		return -float32(32768 - n)/float32(32768)
	}
}
