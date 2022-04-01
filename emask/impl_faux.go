package emask

import "math"
import "math/bits"
import "image"
import "image/draw"

import "golang.org/x/image/vector"
import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/efixed"

// configurable constants
const applyExtraWidthFix = true // bad fixes for bad faux-bold algorithms

// A rasterizer to draw oblique and faux-bold text. Notice that in general
// you should be using the font's italic and bold versions instead of these
// fake effects (specially for bold, faux-bold is really bad and has some
// issues when stems don't reach full opacity).
//
// In 64-bit computers, using the FauxRasterizer without effects performs
// basically the same as DefaultRasterizer. Using reasonable skew factors
// for oblique tends to increase the rasterization time around 15%, and
// using faux-bold increases the rasterization time in ~50%. It depends
// a lot on how extreme the effects are.
//
// This rasterizer was created mostly to serve as an example of how to
// create modified rasterizers, featuring both modification of glyph
// control points (oblique) and post-processing of the generated mask
// (faux-bold).
type FauxRasterizer struct {
	// same fields as DefaultRasterizer
	rasterizer vector.Rasterizer
	onChange func(Rasterizer)
	auxOnChange func(*FauxRasterizer)
	maskAdjust image.Point
	cacheSignature uint64
	normOffsetX float64
	normOffsetY float64
	hasInitSig bool // flag for initialized signature

	skewing float64 // between -1 (45 degrees) and 1 (-45 degrees)
	                // (quantized to be representable without loss
					    //  in 16 bits)

	// extra width (faux-bold) related fields
	xwidth float64
	xwidthFract float64 // the fractional part of xwidth (quantized to 1/64ths)
	xwidthWhole uint16  // the whole part of xwidth
	xwidthTailMod uint16
	xwidthTail []uint8 // internal implementation detail
}

// Sets the oblique skewing factor. Values outside the [-1, 1] range will
// be clamped. -1 corresponds to a counter-clockwise angle of 45 degrees
// from the vertical. 1 corresponds to -45 degrees.
//
// In general, most italic fonts have an italic angle between -6 and -9
// degrees, which would correspond to skew factors in the [0.13, 0.2] range.
func (self *FauxRasterizer) SetSkewFactor(factor float64) {
	// normalize and store new skewing factor
	if factor == 0 {
		if self.skewing == 0 { return }
		self.skewing = 0
		self.cacheSignature = self.cacheSignature & 0xFFFF0FFF0000FFFF
	} else {
		if factor >  1.0 { factor =  1.0 }
		if factor < -1.0 { factor = -1.0 }
		quantized := int32(factor*32768)
		if quantized == 0 {
			if factor > 0 { quantized =  1 }
			if factor < 0 { quantized = -1 }
		}
		newSkewing := float64(quantized)/32768
		if self.skewing == newSkewing { return }
		self.skewing = newSkewing

		// update cache signature
		offset := int32(32768)
		if quantized > 0 { offset = 32767 } // allow reaching 1 skew factor
		sigMark := uint16(quantized + offset)
		self.cacheSignature = self.cacheSignature & 0xFFFF0FFF0000FFFF
		self.cacheSignature |= 0x0000100000000000 // flag for "active italics"
		self.cacheSignature |= uint64(sigMark) << 16
	}

	self.notifyChange()
}

// Gets the skewing factor [-1.0, 1.0] used for the oblique style.
func (self *FauxRasterizer) GetSkewFactor() float64 { return self.skewing }

// Sets the extra width for the faux-bold. Values outside the [0, 1024]
// range will be clamped. Fractional values are allowed, but internally
// the decimal part will be quantized to 1/64ths of a pixel.
//
// Important: when extra width is used for faux-bold, the glyphs will
// become wider. If you want to adapt the positioning of the glyphs to
// account for this widening, you can use an esizer.AdvancePadSizer,
// link the rasterizer to it through SetAuxOnChangeFunc and update
// the padding with the value of GetExtraWidth, for example.
func (self *FauxRasterizer) SetExtraWidth(extraWidth float64) {
	// normalize and store new skewing factor
	if extraWidth <= 0 {
		if self.xwidth == 0 { return } // shortcut
		self.xwidth = 0
		self.xwidthWhole = 0
		self.xwidthFract = 0
		self.cacheSignature = self.cacheSignature & 0xFFF0FFFFFFFF0000
	} else {
		if extraWidth > 1024.0 { extraWidth = 1024 }
		quantized := uint32(extraWidth*64)
		if quantized >= 65536 {
			quantized = 65535
		}
		if quantized == 0 { quantized = 1 }
		newExtraWidth := float64(quantized)/64
		if self.xwidth == newExtraWidth { return } // shortcut
		self.xwidth = newExtraWidth

		// compute whole part for the given extra width
		wholeFloat, fractFloat := math.Modf(self.xwidth)
		self.xwidthWhole = uint16(wholeFloat)
		self.xwidthFract = fractFloat
		if len(self.xwidthTail) < int(self.xwidthWhole) {
			if self.xwidthTail == nil {
				self.xwidthTail = make([]uint8, 8)
				self.xwidthTailMod = 7
			} else {
				targetSize := uint16RoundToNextPow2(self.xwidthWhole)
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
		self.cacheSignature = self.cacheSignature & 0xFFF0FFFFFFFF0000
		self.cacheSignature |= 0x00000B0000000000 // flag for "active bold"
		self.cacheSignature |= uint64(uint16(quantized))
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
func (self *FauxRasterizer) GetExtraWidth() float64 { return self.xwidth }

// Satisfies the UserCfgCacheSignature interface.
func (self *FauxRasterizer) SetHighByte(value uint8) {
	self.cacheSignature &= 0x00FFFFFFFFFFFFFF
	self.cacheSignature |= uint64(value) << 56
	self.notifyChange()
}

// Satisfies the Rasterizer interface. The cache signature for the
// faux rasterizer has the following shape:
//  - 0xFF00000000000000 bits for UserCfgCacheSignature's high byte.
//  - 0x00FF000000000000 bits being 0xFA (self signature byte).
//  - 0x0000F00000000000 bits being 0x1 if italics are enabled.
//  - 0x00000F0000000000 bits being 0xB if bold is enabled.
//  - 0x00000000FFFF0000 bits encoding the skewing [-1, 1] as [0, 65535],
//    with the zero skewing not having a representation here (signatures
//    are still different due to the "italics-enabled" flag).
//  - 0x000000000000FFFF bits encoding the extra bold width in 64ths of
//    a pixel and encoded as a uint16.
func (self *FauxRasterizer) CacheSignature() uint64 {
	// initialize "FA" signature (standing for FAUX) bits if relevant
	if !self.hasInitSig {
		self.hasInitSig = true
		self.cacheSignature |= 0x00FA000000000000
	}

	// return cache signature
	return self.cacheSignature
}

func (self *FauxRasterizer) fixedToFloat32Coords(point fixed.Point26_6) (float32, float32) {
	// apply skewing here!
	fx := float64(point.X)/64
	fy := float64(point.Y)/64
	x := fx - fy*self.skewing + self.normOffsetX
	y := fy + self.normOffsetY
	return float32(x), float32(y)
}

// Satisfies the Rasterizer interface.
func (self *FauxRasterizer) Rasterize(outline sfnt.Segments, fract fixed.Point26_6) (*image.Alpha, error) {
	self.newOutline(outline, fract)
	mask := image.NewAlpha(self.rasterizer.Bounds())
	processOutline(self, outline)
	self.rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	mask.Rect = mask.Rect.Add(self.maskAdjust)

	if self.xwidth > 0 {
		self.applyExtraWidth(mask.Pix, mask.Stride)
	}
	return mask, nil
}

// Like SetOnChangeFunc, but not reserved for internal Renderer use.
// This is provided so you can link a custom Sizer to the Rasterizer and
// get notified when its configuration changes.
func (self *FauxRasterizer) SetAuxOnChangeFunc(onChange func(*FauxRasterizer)) {
	self.auxOnChange = onChange
}

func (self *FauxRasterizer) notifyChange() {
	if self.onChange    != nil { self.onChange(self)    }
	if self.auxOnChange != nil { self.auxOnChange(self) }
}

func (self *FauxRasterizer) newOutline(outline sfnt.Segments, fract fixed.Point26_6) error {
	glyphBounds := outline.Bounds()

	// adjust the bounds accounting for skewing
	if self.skewing != 0 {
		shiftA := efixed.FromFloat64RoundAwayZero((float64(glyphBounds.Min.Y)/64)*self.skewing)
		shiftB := efixed.FromFloat64RoundAwayZero((float64(glyphBounds.Max.Y)/64)*self.skewing)
		if self.skewing >= 0 { // don't make me explain...
			glyphBounds.Min.X -= shiftB
			glyphBounds.Max.X -= shiftA
		} else { // ...I don't actually know what I'm doing
			glyphBounds.Min.X -= shiftA
			glyphBounds.Max.X -= shiftB
		}
	}

	// adjust the bounds accounting for faux-bold extra width
	if self.xwidth > 0 {
		glyphBounds.Max.X += fixed.Int26_6(int32(math.Ceil(self.xwidth)) << 6)
	}

	// similar to default rasterizer
	size, normOffset, adjust := figureOutBounds(glyphBounds, fract)
	self.maskAdjust = adjust
	self.normOffsetX = float64(normOffset.X)/64
	self.normOffsetY = float64(normOffset.Y)/64
	self.rasterizer.Reset(size.X, size.Y)
	self.rasterizer.DrawOp = draw.Src
	return nil
}

// ==== EXTRA WIDTH COMPUTATIONS ====
// I spent like a whole week trying to figure this **** out and fighting
// madness at the depths of test-hell. A few times I thought I was finally
// starting to see the light at the end of the tunnel, but oh boy not so
// fast, the latest test may now be working BUT ALL THE PREVIOUS ONES BROKE.
//
// Today, the code is still bad and miserable. I hate this thing.
// Just use a proper bold font and leave me alone.
//
// More seriously now... Better faux-bold must be done through shape
// expansion, working directly with the outline points, but that's
// very tricky to do (e.g: https://github.com/libass/libass/blob/
// 7bf4bee0fc9a1d6257a105a3c19df6cf08733f8e/libass/ass_outline.c#L499),
// and even freetype implementations are not "perfect".

func (self *FauxRasterizer) applyExtraWidth(pixels []uint8, stride int) {
	// extra width is applied independently to each row
	for x := 0; x < len(pixels); x += stride {
		self.applyRowExtraWidth(pixels[x : x + stride], pixels, x, stride)
	}
}

func (self *FauxRasterizer) applyRowExtraWidth(row []uint8, pixels []uint8, start int, stride int) {
	var peakAlpha uint8

	// for each row, the idea is to ascend to the biggest alpha
	// values first, and then when falling apply the extra width,
	// mostly as a keep-max-of-last-n-alpha-values.
	for index := 0; index < len(row); {
		index, peakAlpha = self.extraWidthRowAscend(row, index)
		if peakAlpha == 0 { return }
		if applyExtraWidthFix {
			peakAlpha = self.peakAlphaFix(row, index, pixels, start, stride, peakAlpha)
		}
		index = self.extraWidthRowFall(row, index, peakAlpha)
	}
}

func (self *FauxRasterizer) peakAlphaFix(row []uint8, index int, pixels []uint8, start int, stride int, peakAlpha uint8) uint8 {
	if peakAlpha == 255 { return peakAlpha }

	// boundaries
	if index < 2 { return peakAlpha }
	if index + 1 >= len(row) { return peakAlpha }
	aboveIndex := (start + index - 1 - stride)
	belowIndex := (start + index - 1 + stride)
	if aboveIndex < 0 || belowIndex > len(pixels) { return peakAlpha }

	// "in stem" heuristic
	upSum := uint16(pixels[aboveIndex - 1]) + uint16(pixels[aboveIndex]) + uint16(pixels[aboveIndex + 1])
	dwSum := uint16(pixels[belowIndex - 1]) + uint16(pixels[belowIndex]) + uint16(pixels[belowIndex + 1])
	min := upSum
	if dwSum < min { min = dwSum }
	if min > 255 { min = 255 }
	min8 := uint8(min)
	if min8 >= peakAlpha {
		peakAlpha = min8
		if row[index - 1] > 0 {
			row[index - 1] = uint8((uint16(peakAlpha) + uint16(row[index - 1]))/2)
		}
	}

	// what about taking the values bigger than us, count how many there are,
	// and apply some normalization based on that? both for the new value,
	// and the original peak..?
	return peakAlpha
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
func (self *FauxRasterizer) extraWidthRowFall(row []uint8, index int, peakAlpha uint8) int {
	// apply the whole width part...
	whole := self.xwidthWhole
	if whole == 0 { // ...unless there's no whole part, I guess
		return self.extraWidthRowFractFall(row, index, peakAlpha)
	}

	for n := uint16(0); n < whole; n++ {
		currAlpha := row[index]
		if currAlpha >= peakAlpha { return index }
		row[index] = peakAlpha
		self.xwidthTail[n] = currAlpha
		index += 1
	}

	// we are done with the whole width peak part. now... what's this?
	mod := self.xwidthTailMod
	if whole > 1 { self.backfixTail(whole) }

	// propagate the tail
	tailIndex   := uint16(0)
	prevAlpha   := peakAlpha
	prevTailAdd := peakAlpha
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
	// do you see the shape of this code? it's obscene.
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
	// I have reasons to believe this operation can't overflow,
	// but don't kill me if it ever does, just report it.
	// Note: I originally used pre-computed tables for the products.
	//       They are indeed generally faster, but for me it wasn't enough
	//       to deserve the extra computations when setting the extra width
	//       (nor the 512 extra bytes of space required).
	// Note2: I also tried to optimize for self.xwidthFract == 0, but it
	//        did not help, neither here nor earlier in the process. Maybe
	//        operating with ones and zeros are fast paths anyway.
	prevWeight := uint8(self.xwidthFract*float64(prevAlpha))
	currWeight := uint8((1 - self.xwidthFract)*float64(currAlpha))
	return prevWeight + currWeight
}

// ==== FROM HERE ON METHODS ARE THE SAME AS DEFAULT RASTERIZER ====
// (...so you can ignore them, nothing new here)

// Satisfies the vectorTracer interface.
func (self *FauxRasterizer) MoveTo(point fixed.Point26_6) {
	x, y := self.fixedToFloat32Coords(point)
	self.rasterizer.MoveTo(x, y)
}

// Satisfies the vectorTracer interface.
func (self *FauxRasterizer) LineTo(point fixed.Point26_6) {
	x, y := self.fixedToFloat32Coords(point)
	self.rasterizer.LineTo(x, y)
}

// Satisfies the vectorTracer interface.
func (self *FauxRasterizer) QuadTo(control, target fixed.Point26_6) {
	cx, cy := self.fixedToFloat32Coords(control)
	tx, ty := self.fixedToFloat32Coords(target)
	self.rasterizer.QuadTo(cx, cy, tx, ty)
}

// Satisfies the vectorTracer interface.
func (self *FauxRasterizer) CubeTo(controlA, controlB, target fixed.Point26_6) {
	cax, cay := self.fixedToFloat32Coords(controlA)
	cbx, cby := self.fixedToFloat32Coords(controlB)
	tx , ty  := self.fixedToFloat32Coords(target)
	self.rasterizer.CubeTo(cax, cay, cbx, cby, tx, ty)
}

// Satisfies the Rasterizer interface.
func (self *FauxRasterizer) SetOnChangeFunc(onChange func(Rasterizer)) {
	self.onChange = onChange
}
