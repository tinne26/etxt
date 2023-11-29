package etxt

import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/cache"
import "github.com/tinne26/etxt/sizer"
import "github.com/tinne26/etxt/fract"

// This file contains the Renderer type definition and all the
// getter and setter methods. Actual operations are split in other
// files.

// The [Renderer] is the heart of etxt and the type around which
// everything else revolves.
//
// Renderers have three groups of functions:
//  - Simple functions to adjust basic text properties like font,
//    size, color, align, etc.
//  - Simple functions to draw and measure text.
//  - Gateways to access more advanced or specific functionality.
//
// Gateways are auxiliary types that group specialized functions together
// and keep them out of the way for most workflows that won't require them.
// The current gateways are the following:
//  - [Renderer.Utils](), to access non-essential but handy functions.
//  - [Renderer.Fract](), to access specialized fractional positioning functionality.
//  - [Renderer.Glyph](), to access low level functions for glyphs and
//    glyph masks.
//  - [Renderer.Twine](), to access rich text and [Twine] functionality.
//
// To create a renderer, using [NewRenderer]() is recommended. Otherwise,
// you will need to set lots of properties manually or depend on
// [RendererUtils.FillMissingProperties]() before you can start using it.
// In all cases, you still need to set a font before drawing or measuring
// text, and in most practical scenarios you will also want to set a cache,
// the text size, the text color and the align.
//
// If you need further help or guidance, I recommend reading ["advice on 
// renderers"] or going through the code on the [examples] folder.
//
// ["advice on renderers"]: https://github.com/tinne26/etxt/blob/main/docs/renderer.md
// [examples]: https://github.com/tinne26/etxt/blob/main/examples
type Renderer struct {
	state restorableState
	restorableStates []restorableState
	
	cacheHandler cache.GlyphCacheHandler
	twineEffectFuncs []TwineEffectFunc
	twineMotionFuncs []TwineMotionFunc
	twineStorage []any
	customDrawFn func(Target, sfnt.GlyphIndex, fract.Point)
	fonts []*sfnt.Font
	buffer sfnt.Buffer
}

// Creates a new [Renderer], intialized with default values.
//
// Setting a font through [Renderer.SetFont]() or [RendererUtils.SetFontBytes]()
// is required before being able to operate with it. It's also heavily
// recommended to set a cache (none by default) right from the start, for
// example using [RendererUtils.SetCache8MiB]().
func NewRenderer() *Renderer {
	// No font sizer change notification required (there's no font yet)
	return &Renderer{
		state: restorableState{
			fontColor: color.RGBA{255, 255, 255, 255},
			fontSizer: &sizer.DefaultSizer{},
			rasterizer: &mask.DefaultRasterizer{},
			horzQuantization: uint8(Qt4th),
			vertQuantization: uint8(QtFull),
			align: Left | Baseline,
			scale: fract.One,
			logicalSize: 16*fract.One,
			scaledSize: 16*fract.One,
		},
		fonts: make([]*sfnt.Font, 0, 1),
	}
}

// Sets the logical font size to be used on subsequent operations.
// Sizes are given in pixels and can't be negative. Maximum size
// is limited around ~16K.
//
// By default, the renderer will draw text at a logical size of 16px.
//
// The relationship between font size and the size of its glyphs
// is complicated and can vary a lot between fonts, but to
// provide a [general reference]:
//  - A capital latin letter is usually around 70% as tall as
//    the given size. E.g.: at 16px, "A" will be 10-12px tall.
//  - A lowercase latin letter is usually around 48% as tall as
//    the given size. E.g.: at 16px, "x" will be 7-9px tall.
//
// See also [Renderer.SetScale]() for proper handling of high
// resolution text and display scaling.
//
// [general reference]: https://github.com/tinne26/etxt/blob/main/docs/px-size.md
func (self *Renderer) SetSize(size float64) {
	// TODO: test with size zero for draws and measures and all that,
	//       as well as fractional but almost zero sizes. the rounding
	//       to zero is reasonable for such extreme cases.
	self.fractSetSize(fract.FromFloat64Up(size))
}

// Returns the current logical font size. The default value is 16.
//
// Notice that the returned value doesn't take scaling into
// account (see [Renderer.SetScale]()).
func (self *Renderer) GetSize() float64 {
	return self.fractGetSize().ToFloat64()
}

// Sets the display scaling factor to be used for the text size
// on subsequent operations.
//
// If you don't know much about display scaling, read [this guide].
// Understanding display scaling is critical to be able to render
// non-crappy text across different devices.
//
// The scale must be non-negative. Its default value is 1.0.
//
// [this guide]: https://github.com/tinne26/etxt/blob/main/docs/display-scaling.md
func (self *Renderer) SetScale(scale float64) {
	self.fractSetScale(fract.FromFloat64Up(scale))
}

// Returns the current display scaling factor used for the
// text as a float64. See [Renderer.SetScale]() for more details.
func (self *Renderer) GetScale() float64 {
	return self.fractGetScale().ToFloat64()
}

// Sets the text direction to be used on subsequent operations.
//
// Do not confuse text direction with horizontal align. Text
// direction is typically only changed for right-to-left languages
// like Arabic, Hebrew or Persian.
//
// By default, the direction is [LeftToRight].
func (self *Renderer) SetDirection(dir Direction) {
	// basically, this can change the text iteration order,
	// from first \n to next, to next \n to first.
	switch dir {
	case LeftToRight, RightToLeft:
		self.state.textDirection = dir
	default:
		panic("invalid direction")
	}
}

// Returns the current main text direction. See [Renderer.SetDirection]()
// for more details.
func (self *Renderer) GetDirection() Direction {
	return self.state.textDirection
}

// Sets the font to be used on subsequent operations. Without a
// font, a renderer is fundamentally useless, so don't forget to
// set this up.
//
// Further pointers and advice:
//  - If you only have the unparsed font file data, consider [RendererUtils.SetFontBytes]().
//  - If you need more robust font management, see [github.com/tinne26/etxt/font.Library].
func (self *Renderer) SetFont(font *sfnt.Font) {
	// ensure there's enough space in the fonts slice
	fontIndex := int(self.state.fontIndex)
	self.fonts = ensureSliceSize(self.fonts, fontIndex + 1)

	// assign font if new
	if font == self.state.activeFont { return }
	self.fonts[fontIndex] = font
	self.state.activeFont = font
	
	// notify font change
	self.notifyFontChange(font)
}

func (self *Renderer) notifyFontChange(font *sfnt.Font) {
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFontChange(font)
	}
	if self.state.fontSizer != nil {
		self.state.fontSizer.NotifyChange(font, &self.buffer, self.state.scaledSize)
	}
}

// Returns the current font. The font is nil by default.
func (self *Renderer) GetFont() *sfnt.Font {
	return self.state.activeFont
}

// Sets the blend mode to be used on subsequent operations.
// The default blend mode will compose glyphs over the active
// target with regular alpha blending.
func (self *Renderer) SetBlendMode(blendMode BlendMode) {
	self.state.blendMode = blendMode
}

// Returns the renderer's [BlendMode]. As far as I know, this is only
// strictly necessary when implementing draw operations with custom
// shaders.
func (self *Renderer) GetBlendMode() BlendMode {
	return self.state.blendMode
}

// Sets the color to be used on subsequent draw operations.
// The default color is white.
func (self *Renderer) SetColor(fontColor color.Color) {
	self.state.fontColor = fontColor
}

// Returns the current drawing color.
func (self *Renderer) GetColor() color.Color {
	return self.state.fontColor
}

// Returns the current [sizer.Sizer].
//
// The most common use of sizers is adjusting line height or glyph
// interspacing. Outside of that, sizers can also be relevant when
// trying to obtain information about font metrics or when making
// custom glyph mask rasterizers, but it's fairly uncommon for the
// average user to have to worry about all these things.
func (self *Renderer) GetSizer() sizer.Sizer {
	return self.state.fontSizer
}

// Sets the sizer to be used on subsequent operations.
//
// The most common use of sizers is adjusting line height or glyph
// interspacing. Outside of that, sizers can also be relevant when
// trying to obtain information about font metrics or when making
// custom glyph mask rasterizers, but it's fairly uncommon for the
// average user to have to worry about all these things.
func (self *Renderer) SetSizer(fontSizer sizer.Sizer) {
	if self.state.fontSizer == fontSizer { return }
	self.state.fontSizer = fontSizer
	self.state.fontSizer.NotifyChange(self.state.activeFont, &self.buffer, self.state.scaledSize)
}

// Returns the current glyph cache handler, which is nil by default.
//
// Rarely used unless you are examining the cache handler manually.
func (self *Renderer) GetCacheHandler() cache.GlyphCacheHandler {
	return self.cacheHandler
}

// Sets the glyph cache handler used by the renderer. By default,
// no cache is used, but you almost always want to set one.
//
// The easiest way is to use [RendererUtils.SetCache8MiB](), but that's
// not suitable for all use-cases. The general approach is to create
// a cache manually, obtain a cache handler from it and set it:
//   glyphsCache := cache.NewDefaultCache(16*1024*1024) // 16MiB cache
//   renderer.SetCacheHandler(glyphsCache.NewHandler())
// See [cache.NewDefaultCache]() for more details.
//
// A cache handler can only be used with a single renderer, but you
// may create multiple handlers from the same underlying cache and
// use them with multiple renderers.
func (self *Renderer) SetCacheHandler(cacheHandler cache.GlyphCacheHandler) {
	self.cacheHandler = cacheHandler
	if cacheHandler == nil {
		if self.state.rasterizer != nil { self.state.rasterizer.SetOnChangeFunc(nil) }
		return
	}

	if self.state.rasterizer != nil {
		self.state.rasterizer.SetOnChangeFunc(cacheHandler.NotifyRasterizerChange)
	}

	cacheHandler.NotifySizeChange(self.state.scaledSize)
	font := self.GetFont()
	if font != nil { cacheHandler.NotifyFontChange(font) }
	if self.state.rasterizer != nil {
		cacheHandler.NotifyRasterizerChange(self.state.rasterizer)
	}
}

// Exposes the renderer's internal [*sfnt.Buffer].
// Only exposed for advanced interaction with the sfnt package
// or the [sizer.Sizer] interface.
func (self *Renderer) GetBuffer() *sfnt.Buffer {
	return &self.buffer
}
