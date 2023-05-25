package etxt

import "strconv"

import "github.com/tinne26/etxt/fract"

// Quantization levels for [RendererFract.SetQuantization]().
//
// Only the equispaced quantization values are given. Other values like
// [fract.Unit](22) (which approximates one third of a pixel, ceil(64/3))
// could work too, but they all result in potentially uneven distributions of
// the glyph positions. Such positions would make the results of text measuring
// functions dependent on the text direction, align and starting position, which
// would make centering operations and the API impractically complicated.
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

// Same as [Renderer.SetScale](), but avoiding a conversion from float64
// to [fract.Unit].
func (self *RendererFract) SetScale(scale fract.Unit) {
	(*Renderer)(self).fractSetScale(scale)
}

// Fractional version of [Renderer.GetScale]().
func (self *RendererFract) GetScale() fract.Unit {
	return (*Renderer)(self).fractGetScale()
}

// Sets the quantization levels to be used on subsequent operations.
// Recommended values are [QtNone], [Qt4th], [QtHalf], [QtFull] and
// the other existing constants.
//
// Non-equispaced values are technically allowed but are not recommended,
// as drawing and measuring algorithms may break in subtle ways in
// different edge cases.
//
// By default, quantization is fully enabled ([QtFull], [QtFull]).
//
// Values below 1 or above 64 fractional units will panic.
func (self *RendererFract) SetQuantization(horz, vert fract.Unit) {
	(*Renderer)(self).fractSetQuantization(horz, vert)
}

// Returns the current quantization levels.
// See [RendererFract.SetQuantization]() for more context.
func (self *RendererFract) GetQuantization() (horz, vert fract.Unit) {
	return (*Renderer)(self).fractGetQuantization()
}

func (self *RendererFract) MeasureHeight(text string) fract.Unit {
	return (*Renderer)(self).fractMeasureHeight(text)
}

func (self *RendererFract) Draw(target TargetImage, text string, x, y fract.Unit) {
	(*Renderer)(self).fractDraw(target, text, x, y)
}

func (self *RendererFract) DrawWithWrap(target TargetImage, text string, x, y fract.Unit, widthLimit int) {
	(*Renderer)(self).fractDrawWithWrap(target, text, x, y, widthLimit)
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
	if self.missingBasicProps() { self.initBasicProps() }
	if self.logicalSize == size { return }
	self.logicalSize = size
	self.refreshScaledSize()
}

func (self *Renderer) fractGetSize() fract.Unit {
	if self.missingBasicProps() { self.initBasicProps() }
	return self.logicalSize
}

func (self *Renderer) fractSetScale(scale fract.Unit) {
	// safety check
	if scale < 0 { panic("negative scaling factor") }

	// set new scale
	if self.missingBasicProps() { self.initBasicProps() }
	if self.scale == scale { return }
	self.scale = scale
	self.refreshScaledSize()
}

func (self *Renderer) fractGetScale() fract.Unit {
	if self.missingBasicProps() { self.initBasicProps() }
	return self.scale
}

// Must be called after logical size or scale changes.
func (self *Renderer) refreshScaledSize() {
	scaledSize := self.scaleLogicalSize(self.logicalSize)
	
	if scaledSize == self.scaledSize { return } // yeah, not likely
	self.scaledSize = scaledSize

	// notify changes
	if self.cacheHandler != nil {
		self.cacheHandler.NotifySizeChange(self.scaledSize)
	}
	if self.fontSizer != nil {
		self.fontSizer.NotifyChange(self.GetFont(), &self.buffer, self.scaledSize)
	}
}

func (self *Renderer) fractSetQuantization(horz, vert fract.Unit) {
	if horz > 64 || horz < 1 || vert > 64 || vert < 1 {
		panic("quantization levels must be in the [1, 64] range")
	}
	self.horzQuantization = uint8(horz)
	self.vertQuantization = uint8(vert)
}

func (self *Renderer) fractGetQuantization() (horz, vert fract.Unit) {
	return fract.Unit(self.horzQuantization), fract.Unit(self.vertQuantization)
}
