package etxt

import "strconv"

import "github.com/tinne26/etxt/fract"

// [Gateway] to [RendererFract] functionality.
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.6#Renderer
func (self *Renderer) Fract() *RendererFract {
	return (*RendererFract)(self)
}

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to perform operations with fractional units.
//
// Fractional units allow us to operate with a higher level of precision
// when drawing or measuring text. The use-cases for this are rather
// limited, though; as a rule of thumb, ignore these advanced features 
// unless you find yourself really needing them.
//
// In general, this type is used through method chaining:
//   renderer.Fract().Draw(canvas, text, x, y)
//
// All the fractional operations depend on the [fract.Unit] type, so
// make sure to check out the [fract] subpackage if you need more context
// to understand how everything ties together.
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.6#Renderer
type RendererFract Renderer

// ---- wrapper methods ----

// Fractional and lower level version of [Renderer.SetSize]().
func (self *RendererFract) SetSize(size fract.Unit) {
	(*Renderer)(self).fractSetSize(size)
}

// Fractional and lower level version of [Renderer.GetSize]().
func (self *RendererFract) GetSize() fract.Unit {
	return (*Renderer)(self).fractGetSize()
}

// Fractional and lower level version of [Renderer.SetScale]().
func (self *RendererFract) SetScale(scale fract.Unit) {
	(*Renderer)(self).fractSetScale(scale)
}

// Fractional and lower level version of [Renderer.GetScale]().
func (self *RendererFract) GetScale() fract.Unit {
	return (*Renderer)(self).fractGetScale()
}

// Returns the scaled text size (logicalSize*scale).
// 
// Having access to the renderer's scaled font size
// is useful when working with sizers and trying to
// obtain specific metrics for advanced use-cases.
func (self *RendererFract) GetScaledSize() fract.Unit {
	return self.state.scaledSize
}

// Sets the horizontal quantization level to be used on subsequent
// operations. Valid values are limited to the existing Qt constants
// (e.g. [QtNone], [QtFull], [QtHalf]).
//
// By default, [NewRenderer]() initializes the horizontal quantization
// to [Qt4th]. This is generally a reasonable compromise between quality
// and performance... unless you are using pixel-art-like fonts; in that
// case, setting the quantization to [QtFull] is much preferable.
//
// See also [RendererFract.SetVertQuantization]().
func (self *RendererFract) SetHorzQuantization(horz fract.Unit) {
	(*Renderer)(self).fractSetHorzQuantization(horz)
}

// Sets the vertical quantization level to be used on subsequent
// operations. Valid values are limited to the existing Qt constants
// (e.g. [Qt4th], [Qt8th], [Qt16th]).
//
// By default, [NewRenderer]() initializes the vertical quantization
// to [QtFull]. Most languages are written horizontally, so you almost
// never want to pay the price for high vertical positioning resolution.
//
// See also [RendererFract.SetHorzQuantization]().
func (self *RendererFract) SetVertQuantization(horz fract.Unit) {
	(*Renderer)(self).fractSetVertQuantization(horz)
}

// Returns the current horizontal and vertical quantization levels.
func (self *RendererFract) GetQuantization() (horz, vert fract.Unit) {
	return (*Renderer)(self).fractGetQuantization()
}

// func (self *RendererFract) MeasureHeight(text string) fract.Unit {
// 	return (*Renderer)(self).fractMeasureHeight(text)
// }

// Fractional and lower level version of [Renderer.Draw]().
func (self *RendererFract) Draw(target Target, text string, x, y fract.Unit) {
	(*Renderer)(self).fractDraw(target, text, x, y)
}

// Fractional and lower level version of [Renderer.DrawWithWrap]().
func (self *RendererFract) DrawWithWrap(target Target, text string, x, y fract.Unit, widthLimit int) {
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
