package etxt

import "strconv"

import "github.com/tinne26/etxt/fract"

// Quantization levels for [RendererFract.SetQuantization]().
//
// Only the perfectly equidistant quantization values are given. Other
// values like fract.Unit(22) (~one third of a pixel = ceil(64/3)) could
// work too, but they all result in potentially uneven distributions of
// the glyph positions. These would make the results of text measuring
// functions dependent on the text direction, align and working position,
// which would make centering impractically complicated. The API would
// also become exceedingly difficult to use correctly.
const (
	QtNone = fract.Unit( 1) // full glyph position resolution (1/64ths of a pixel)
	Qt32th = fract.Unit( 2) // quantize glyph positions to 1/32ths of a pixel
	Qt16th = fract.Unit( 4) // quantize glyph positions to 1/16ths of a pixel
	Qt8th  = fract.Unit( 8) // quantize glyph positions to 1/ 8ths of a pixel
	Qt4th  = fract.Unit(16) // quantize glyph positions to 1/ 4ths of a pixel
	QtHalf = fract.Unit(32) // quantize glyph positions to half of a pixel
	QtFull = fract.Unit(64) // full glyph position quantization (default)
)

// Access the renderer in [RendererFract] mode. This mode allows you to 
// configure or operate the renderer with an increased level of precision, up
// to 1/64th of a pixel.
func (self *Renderer) Fract() *RendererFract {
	return (*RendererFract)(self)
}

// A wrapper type for using a [Renderer] in "fractional mode". This mode 
// allows you to configure or operate the renderer with an increased level 
// of precision, up to 1/64th of a pixel.
//
// Fractional operations are typically relevant when animating or trying to
// respect the text flow with the highest precision possible.
//
// The fractional getters and setters can also be useful when saving state
// of the renderer to be restored later, as floating point conversions can
// be avoided.
//
// All the fractional operations depend on the [fract.Unit] type, so make
// sure to check out the [etxt/fract] subpackage if you need more context
// to understand how everything ties together.
//
// Notice that this type exists almost exclusively for documentation and
// structuring purposes. To most effects, you could consider the methods
// part of [Renderer] itself.
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

// Same as [Renderer.SetScale](), but avoids a conversion from float64
// to fract.Unit.
func (self *RendererFract) SetScale(scale fract.Unit) {
	(*Renderer)(self).fractSetScale(scale)
}

// Fractional version of [Renderer.GetScale]().
func (self *RendererFract) GetScale() fract.Unit {
	return (*Renderer)(self).fractGetScale()
}

// Sets the renderer's quantization level. You should use [QtNone],
// [Qt4th], [QtHalf], [QtFull] and the other existing constants. As
// their documentation explains, other arbitrary values may get you
// in trouble.
//
// By default, quantization is fully enabled ([QtFull], [QtFull]).
//
// Values below one or above 64 fractional units will panic.
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

func (self *RendererFract) Draw(target TargetImage, text string, x, y fract.Unit) fract.Point {
	return (*Renderer)(self).fractDraw(target, text, x, y)
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
	scaledSize := self.logicalSize.MulDown(self.scale) // *
	// * We use MulDown here to compensate having used FromFloat64Up()
	//   on both size and scale conversions. It's not a big deal in
	//   either case, but I guess this reduces the maximum potential
	//   error.
	if scaledSize == self.scaledSize { return } // yeah, not likely
	self.scaledSize = scaledSize

	// notify changes
	if self.cacheHandler != nil {
		self.cacheHandler.NotifySizeChange(self.scaledSize)
	}
	if self.fontSizer != nil {
		self.fontSizer.NotifyChange(self.GetFont(), &self.Buffer, self.scaledSize)
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

