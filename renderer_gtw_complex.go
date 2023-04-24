package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/sizer"

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
	return (*Renderer)(self).complexGetDirection()
}

// Sets the renderer's fonts. Having multiple fonts is mostly relevant
// when working with rich text, for example when trying to combine regular,
// bold and italic subfamilies in the same draw call.
//
// To change the main font statically, use [RendererComplex.SetFontIndex]().
func (self *RendererComplex) SetFonts(fonts []*sfnt.Font) {
	(*Renderer)(self).complexSetFonts(fonts)
}

// Sets the glyph mask rasterizer to be used on subsequent operations.
// Nil rasterizers are not allowed.
func (self *RendererComplex) SetRasterizer(rasterizer mask.Rasterizer) {
	(*Renderer)(self).complexSetRasterizer(rasterizer)
}

// Returns the current glyph mask rasterizer.
//
// This function is only useful when working with configurable rasterizers;
// ignore it if you are using the default glyph mask rasterizer.
//
// Mask rasterizers are not concurrent-safe, so be careful with
// what you do and where you put them.
func (self *RendererComplex) GetRasterizer() mask.Rasterizer {
	return (*Renderer)(self).complexGetRasterizer()
}

// Returns the renderer fonts available for use with rich text.
func (self *RendererComplex) GetFonts() []*sfnt.Font {
	return (*Renderer)(self).fonts
}

// Returns the current [sizer.Sizer].
//
// The most common use of sizers is adjusting line height or glyph
// interspacing. Outside of that, sizers can also be relevant when
// trying to obtain information about font metrics or when making
// custom glyph mask rasterizers, but it's fairly uncommon for the
// average user to have to worry about all these things.
func (self *RendererComplex) GetSizer() sizer.Sizer {
	return (*Renderer)(self).complexGetSizer()
}

// Sets the sizer to be used on subsequent operations. Nil sizers are
// not allowed.
//
// The most common use of sizers is adjusting line height or glyph
// interspacing. Outside of that, sizers can also be relevant when
// trying to obtain information about font metrics or when making
// custom glyph mask rasterizers, but it's fairly uncommon for the
// average user to have to worry about all these things.
func (self *RendererComplex) SetSizer(fontSizer sizer.Sizer) {
	(*Renderer)(self).complexSetSizer(fontSizer)
}

// Sets the active font index to the given value.
// Returns the newly active font.
func (self *RendererComplex) SetFontIndex(index uint8) *sfnt.Font {
	return (*Renderer)(self).complexSetFontIndex(index)
}

// Returns the index of the renderer's main font.
func (self *RendererComplex) GetFontIndex() int {
	return int((*Renderer)(self).fontIndex)
}

// Same as [Renderer.Draw](), but expecting a [Text] instead of a string.
func (self *RendererComplex) Draw(screen TargetImage, text Text, x, y fract.Unit) {
	panic("unimplemented / TODO")
}

// Same as [Renderer.DrawWithWrap](), but expecting a [Text] instead of a string.
func (self *RendererComplex) DrawWithWrap(screen TargetImage, text Text, x, y, widthLimit fract.Unit) {
	panic("unimplemented / TODO")
}

// Exposes the renderer's internal [*sfnt.Buffer].
// Only exposed for advanced interaction with the sfnt package.
func (self *RendererComplex) GetBuffer() *sfnt.Buffer {
	return &((*Renderer)(self).buffer)
}

// ---- implementations ----

func (self *Renderer) complexSetDirection(dir Direction) {
	// basically, this can change the text iteration order,
	// from first \n to next, to next \n to first.
	switch dir {
	case LeftToRight:
		self.internalFlags &= ^internalFlagDirIsRTL
	case RightToLeft:
		self.internalFlags |= internalFlagDirIsRTL
	default:
		panic("invalid direction")
	}
}

func (self *Renderer) complexGetDirection() Direction {
	if self.internalFlags & internalFlagDirIsRTL != 0 {
		return RightToLeft
	} else {
		return LeftToRight
	}
}

func (self *Renderer) complexGetRasterizer() mask.Rasterizer {
	self.initRasterizer()
	return self.rasterizer
}

func (self *Renderer) initRasterizer() {
	if self.internalFlags & internalFlagRasterizer == 0 {
		self.internalFlags |= internalFlagRasterizer
		self.rasterizer = &mask.DefaultRasterizer{}
	}
}

func (self *Renderer) complexSetRasterizer(rasterizer mask.Rasterizer) {
	// assertion
	if rasterizer == nil { panic("nil rasterizers not allowed") }

	// clear rasterizer onChangeFunc
	if self.rasterizer != nil {
		self.rasterizer.SetOnChangeFunc(nil)
	}

	// set rasterizer and mark it as initialized
	self.rasterizer = rasterizer
	self.internalFlags |= internalFlagRasterizer

	// link new rasterizer to the cache handler
	if self.missingBasicProps() { self.initBasicProps() }
	if self.cacheHandler == nil {
		rasterizer.SetOnChangeFunc(nil)
	} else {
		rasterizer.SetOnChangeFunc(self.cacheHandler.NotifyRasterizerChange)
		self.cacheHandler.NotifyRasterizerChange(rasterizer)
	}
}

func (self *Renderer) complexGetSizer() sizer.Sizer {
	self.initSizer()
	return self.fontSizer
}

func (self *Renderer) initSizer() {
	if self.internalFlags & internalFlagSizer == 0 {
		self.internalFlags |= internalFlagSizer
		self.fontSizer = &sizer.DefaultSizer{}
	}
}

func (self *Renderer) complexSetSizer(fontSizer sizer.Sizer) {
	self.fontSizer = fontSizer
	self.internalFlags |= internalFlagSizer
}

func (self *Renderer) complexSetFonts(fonts []*sfnt.Font) {
	oldFont := self.GetFont()
	self.fonts = fonts
	newFont := self.GetFont()
	if oldFont != newFont {
		self.notifyFontChange(newFont)
	}
}

func (self *Renderer) complexSetFontIndex(index uint8) *sfnt.Font {
	// change active font index and notify change if relevant
	oldFont := self.GetFont()
	self.fontIndex = uint8(index)
	newFont := self.GetFont()
	if oldFont != newFont {
		self.notifyFontChange(newFont)
	}

	// return the new selected font
	return newFont
}

// func (self *Renderer) complexSetFont(font *sfnt.Font, key uint8) {
// 	ikey := int(key)
// 	if len(self.fonts) <= ikey {
// 		if cap(self.fonts) <= ikey {
// 			newFonts := make([]*sfnt.Font, ikey + 1)
// 			copy(newFonts, self.fonts)
// 			self.fonts = newFonts
// 		} else {
// 			self.fonts = self.fonts[ : ikey + 1]
// 		}
// 	}

// 	self.fonts[ikey] = font
// }
