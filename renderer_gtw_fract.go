package etxt

import "strconv"

import "github.com/tinne26/etxt/fract"

// Quantization levels for [RendererFract.SetQuantization]().
//
// Only the equispaced quantization values are given. Other values like
// [fract.Unit](22) (which approximates one third of a pixel, ceil(64/3))
// could also work in theory, but in practice they lead to all kinds of
// complications that are simply not worth it.
const (
	QtNone = fract.Unit( 1) // full glyph position resolution (1/64ths of a pixel)
	Qt32th = fract.Unit( 2) // quantize glyph positions to 1/32ths of a pixel
	Qt16th = fract.Unit( 4) // quantize glyph positions to 1/16ths of a pixel
	Qt8th  = fract.Unit( 8) // quantize glyph positions to 1/ 8ths of a pixel
	Qt4th  = fract.Unit(16) // quantize glyph positions to 1/ 4ths of a pixel
	QtHalf = fract.Unit(32) // quantize glyph positions to half of a pixel
	QtFull = fract.Unit(64) // full glyph position quantization (default)
)

// [Gateway] to [RendererFract] functionality.
//
// [Gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
func (self *Renderer) Fract() *RendererFract {
	return (*RendererFract)(self)
}

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to operate a [Renderer] with fractional units.
//
// Fractional units give us an increased level of precision when
// drawing or measuring text. This is typically relevant when animating
// or trying to respect the text flow with the highest precision 
// possible.
//
// In general, this type is used through method chaining:
//   renderer.Fract().Draw(canvas, text, x, y)
//
// The fractional getters and setters can also be useful when saving state
// of the renderer to be restored later, avoiding floating point conversions.
//
// All the fractional operations depend on the [fract.Unit] type, so make
// sure to check out the [etxt/fract] subpackage if you need more context
// to understand how everything ties together.
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
type RendererFract Renderer

// ---- wrapper methods ----

// Fractional version of [Renderer.SetSize]().
func (self *RendererFract) SetSize(size fract.Unit) {
	(*Renderer)(self).fractSetSize(size)
}

// Fractional version of [Renderer.GetSize]().
func (self *RendererFract) GetSize() fract.Unit {
	return (*Renderer)(self).fractGetSize()
}

// Returns the scaled text size (logicalSize*scale).
// 
// This method is only intended be used when you want to
// access specific metrics through the renderer's sizer.
func (self *RendererFract) GetScaledSize() fract.Unit {
	return self.state.scaledSize
}

// Same as [Renderer.SetScale](), but avoiding a conversion from float64
// to [fract.Unit].
func (self *RendererFract) SetScale(scale fract.Unit) {
	(*Renderer)(self).fractSetScale(scale)
}

// Fractional version of [Renderer.GetScale]().
func (self *RendererFract) GetScale() fract.Unit {
	return (*Renderer)(self).fractGetScale()
}

// Sets the horizontal quantization level to be used on subsequent
// operations. Valid values are limited to the existing Qt constants
// (e.g. [QtNone], [QtFull], [QtHalf]).
//
// By default, the horizontal quantization is [Qt4th].
func (self *RendererFract) SetHorzQuantization(horz fract.Unit) {
	(*Renderer)(self).fractSetHorzQuantization(horz)
}

// Sets the vertical quantization level to be used on subsequent
// operations. Valid values are limited to the existing Qt constants
// (e.g. [Qt4th], [Qt8th], [Qt16th]).
//
// By default, the vertical quantization is [QtFull].
func (self *RendererFract) SetVertQuantization(horz fract.Unit) {
	(*Renderer)(self).fractSetVertQuantization(horz)
}

// Returns the current horizontal and vertical quantization levels.
// See [RendererFract.SetQuantization]() for more context.
func (self *RendererFract) GetQuantization() (horz, vert fract.Unit) {
	return (*Renderer)(self).fractGetQuantization()
}

// func (self *RendererFract) MeasureHeight(text string) fract.Unit {
// 	return (*Renderer)(self).fractMeasureHeight(text)
// }

func (self *RendererFract) Draw(target TargetImage, text string, x, y fract.Unit) {
	(*Renderer)(self).fractDraw(target, text, x, y)
}

func (self *RendererFract) DrawWithWrap(target TargetImage, text string, x, y fract.Unit, widthLimit int) {
	(*Renderer)(self).fractDrawWithWrap(target, text, x, y, fract.FromInt(widthLimit))
}

// ---- underlying implementations ----

func (self *Renderer) fractSetSize(size fract.Unit) {
	// range checks
	if size < 0 { panic("negative text size") }
	if size > 0x000F_FFFF { // maximum 12 bits for the size (~16k max size)
		panic("size " + strconv.FormatFloat(size.ToFloat64(), 'f', -1, 64) + " too big")
	}
	// (we are artificially limiting sizes so glyphs don't take
	// more than ~1GB on ebiten or ~0.25GB as alpha images. Even
	// at those levels most computers will choke to death if they
	// try to render multiple characters, but... I tried...)

	// set the new size
	if self.state.logicalSize == size { return }
	self.state.logicalSize = size
	self.refreshScaledSize()
}

func (self *Renderer) fractGetSize() fract.Unit {
	return self.state.logicalSize
}

func (self *Renderer) fractSetScale(scale fract.Unit) {
	// safety check
	if scale < 0 { panic("negative scaling factor") }

	// set new scale
	if self.state.scale == scale { return }
	self.state.scale = scale
	self.refreshScaledSize()
}

func (self *Renderer) fractGetScale() fract.Unit {
	return self.state.scale
}

// Must be called after logical size or scale changes.
func (self *Renderer) refreshScaledSize() {
	scaledSize := self.scaleLogicalSize(self.state.logicalSize)
	
	if scaledSize == self.state.scaledSize { return } // yeah, not likely
	self.state.scaledSize = scaledSize

	// notify changes
	if self.cacheHandler != nil {
		self.cacheHandler.NotifySizeChange(self.state.scaledSize)
	}
	if self.state.fontSizer != nil {
		self.state.fontSizer.NotifyChange(self.GetFont(), &self.buffer, self.state.scaledSize)
	}
}

func (self *Renderer) fractSetHorzQuantization(horz fract.Unit) {
	validateQuantizationValue(horz)
	self.state.horzQuantization = uint8(horz)
}

func (self *Renderer) fractSetVertQuantization(vert fract.Unit) {
	validateQuantizationValue(vert)
	self.state.vertQuantization = uint8(vert)
}

func validateQuantizationValue(value fract.Unit) {
	switch value {
	case QtNone, Qt32th, Qt16th, Qt8th, Qt4th, QtHalf, QtFull:
		// ok
	default:
		panic("invalid quantization value (tip: Qt* constants)")
	}
}

func (self *Renderer) fractGetQuantization() (horz, vert fract.Unit) {
	return fract.Unit(self.state.horzQuantization), fract.Unit(self.state.vertQuantization)
}
