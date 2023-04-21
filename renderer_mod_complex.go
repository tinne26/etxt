package etxt

import "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/fract"

// Also, on the topic of complex scripts and bidi and stuff:
// - allow horizontal align that's Base for following the current base direction.
//   each paragraph may have a different Base direction, though I may not support
//   that directly, it's not that important probably.
// - chars are stored in the order they are typed. makes sense.
// - separate runs for Text kiiinda become necessary. though I can also assume
//   directly from the []byte data with a special separator.
// - punctuation with neutral dir between different dir chars gets base dir direction.
// - actually, if the neutral chars like parens are handled by the Text encoding
//   (¿is that even possible?), the base dir may be irrelevant for that? this means
//   that the main shaping process needs to be aware of this base direction thingie.
//   which seems ok enough. it's basically getting a []Paragraph, where each
//   paragraph is Text with a base direction to apply. good enough for me I guess.
// - yeah, direction doesn't really change anything at runtime, must be known and
//   considered when "compiling" the text. fair enough. also, runs will simply be
//   implicit unless an active run specification is found. but that would mean I
//   have to detect the end... so no point in indicating an explicit length. just
//   change the run direction explicitly when necessary.

// A wrapper type for using a [Renderer] in "complex mode". This mode allows
// accessing advanced renderer properties, as well as operating directly with
// glyphs and the more flexible [Text] type. These types are relevant when
// working with [complex scripts and text shaping], among others.
//
// Notice that this type exists almost exclusively for documentation and
// structuring purposes. To most effects, you could consider the methods
// part of [Renderer] itself.
//
// [complex scripts and text shaping]: https://github.com/tinne26/etxt/blob/main/docs/shaping.md
type RendererComplex Renderer

// Access the renderer in [RendererComplex] mode. This mode allows accessing
// advanced renderer properties, as well as operating directly with glyphs and
// the more flexible [Text] type.
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

func (self *RendererComplex) GetDirection() Direction {
	return (*Renderer)(self).complexGetDirection()
}

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
	(*Renderer)(self).complexGetRasterizer(rasterizer)
}

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

// Sets the current [sizer.Sizer], which must be non-nil.
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

func (self *RendererComplex) GetFontIndex() int {
	return int((*Renderer)(self).fontIndex)
}

func (self *RendererComplex) Draw(screen TargetImage, text Text, x, y fract.Unit) fract.Point {
	// TODO
	return fract.Point{}
}

func (self *RendererComplex) DrawInRect(screen TargetImage, text Text, x, y fract.Unit, opts *RectOptions) fract.Point {
	// TODO
	return fract.Point{}
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
