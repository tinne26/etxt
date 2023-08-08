package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to access advanced [Renderer] properties and
// operating directly with glyphs and the more flexible [Text] type.
//
// These types and features are mainly relevant when working with
// rich text, [complex scripts] and text shaping.
//
// In general, this type is used through method chaining:
//   renderer.Complex().Draw(canvas, text, x, y)
//
// [complex scripts]: https://github.com/tinne26/etxt/blob/main/docs/shaping.md
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
type RendererComplex Renderer

// [Gateway] to [RendererComplex] functionality.
//
// [Gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
func (self *Renderer) Complex() *RendererComplex {
	return (*RendererComplex)(self)
}

// Sets the text direction to be used on subsequent operations.
//
// Do not confuse text direction with horizontal align. Text
// direction is typically only changed for right-to-left languages
// like Arabic, Hebrew or Persian.
//
// By default, the direction is [LeftToRight].
func (self *RendererComplex) SetDirection(dir Direction) {
	(*Renderer)(self).complexSetDirection(dir)
}

// Returns the current main text direction. See [RendererComplex.SetDirection]()
// for more details.
func (self *RendererComplex) GetDirection() Direction {
	return self.state.textDirection
}

// Registers the given font at the given index. Having multiple fonts
// is relevant when working with rich text, for example when trying to
// combine regular, bold and italic subfamilies in the same draw call.
//
// To change the active font, use [RendererComplex.SetFontIndex]().
// The more basic [Renderer.SetFont]() method operates exclusively
// on the active font.
//
// Additional technical details:
//  - Registering a font on the active font index is safe and equivalent
//    to [Renderer.SetFont]().
//  - There is no way to release the underlying font slice except letting
//    the whole renderer be garbage collected.
//  - Setting a nil font to an index beyond the bounds of the underlying
//    slice won't panic or be ignored, it will make the slice grow.
func (self *RendererComplex) RegisterFont(font *sfnt.Font, index FontIndex) {
	(*Renderer)(self).complexRegisterFont(font, index)
}

// Returns the renderer fonts available for use with rich text.
// 
// The returned slice should be treated as read only. At the very
// least, know that modifying the active font externally will
// leave it unsynced with the renderer's cache handler and sizer.
func (self *RendererComplex) GetFonts() []*sfnt.Font {
	return self.fonts
}

// Sets the active font index to the given value. If the index
// exceeds the bounds of the underlying slice, the slice will be
// resized to make the index referenceable.
func (self *RendererComplex) SetFontIndex(index FontIndex) {
	(*Renderer)(self).complexSetFontIndex(index)
}

// Returns the index of the renderer's main font.
func (self *RendererComplex) GetFontIndex() FontIndex {
	return self.state.fontIndex
}

// Same as [Renderer.Draw](), but expecting a [Text] value instead of a string.
func (self *RendererComplex) Draw(target TargetImage, text Text, x, y int) {
	self.FractDraw(target, text, fract.FromInt(x), fract.FromInt(y))
}

// Same as [RendererFract.Draw](), but expecting a [Text] value instead of a string.
//
// This is the fractional version of [RendererComplex.Draw]().
func (self *RendererComplex) FractDraw(target TargetImage, text Text, x, y fract.Unit) {
	panic("unimplemented / TODO")
}

// Same as [Renderer.DrawWithWrap](), but expecting a [Text] value instead of a string.
func (self *RendererComplex) DrawWithWrap(target TargetImage, text Text, x, y int, widthLimit int) {
	self.FractDrawWithWrap(target, text, fract.FromInt(x), fract.FromInt(y), widthLimit)
}

// Same as [RendererFract.DrawWithWrap](), but expecting a [Text] value instead of a string.
//
// This is the fractional version of [RendererComplex.DrawWithWrap]().
func (self *RendererComplex) FractDrawWithWrap(target TargetImage, text Text, x, y fract.Unit, widthLimit int) {
	panic("unimplemented / TODO")
}

// ---- implementations ----

func (self *Renderer) complexSetDirection(dir Direction) {
	// basically, this can change the text iteration order,
	// from first \n to next, to next \n to first.
	switch dir {
	case LeftToRight, RightToLeft:
		self.state.textDirection = dir
	default:
		panic("invalid direction")
	}
}

func (self *Renderer) complexRegisterFont(font *sfnt.Font, index FontIndex) {
	if index == self.state.fontIndex {
		self.SetFont(font)
	} else {
		fontIndex := int(self.state.fontIndex)
		self.fonts = ensureSliceSize(self.fonts, fontIndex + 1)
		self.fonts[fontIndex] = font
	}
}

func (self *Renderer) complexSetFontIndex(index FontIndex) {
	// grow the slice if necessary (as per spec)
	self.fonts = ensureSliceSize(self.fonts, int(index) + 1)

	// change active font index and notify change if relevant
	self.state.fontIndex = index
	newFont := self.fonts[index]
	if newFont != self.state.activeFont {
		self.state.activeFont = newFont
		self.notifyFontChange(newFont)
	}
}
