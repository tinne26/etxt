package etxt

import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/cache"
import "github.com/tinne26/etxt/sizer"
import "github.com/tinne26/etxt/fract"

// Initialization flags used to determine which fields of the *Renderer 
// are ready to be used or still need to be manually initialized. This
// allows the zero value of the renderer to be as valid as possible.
const (
	internalFlagRasterizer uint8 = 0b00000001
	internalFlagSizer      uint8 = 0b00000010
	internalFlagCache      uint8 = 0b00000100 // TODO: implement with pkgNoCacheHandler
	internalFlagBasicProps uint8 = 0b00001000
	internalFlagDirIsRTL   uint8 = 0b10000000 // by default it's LTR
)

// This file contains the Renderer type definition and all the
// getter and setter methods. Actual operations are split in other
// files.

// The [Renderer] is the main type for drawing text provided by etxt.
//
// Renderers allow you to control font, text size, color, text
// alignment and more from a single place.
//
// The zero value is valid, but you must set a font (e.g. [Renderer.SetFont]())
// if you want to do anything useful with it. In most cases, you will
// also want to set a cache, the text size, the color and the align.
type Renderer struct {
	fonts []*sfnt.Font
	gfxFuncs []func(*Renderer, TargetImage, fract.Rect, uint16)

	buffer sfnt.Buffer

	fontColor color.Color
	fontSizer sizer.Sizer
	rasterizer mask.Rasterizer
	cacheHandler cache.GlyphCacheHandler

	internalFlags uint8 // see internalFlag* constants
	horzQuantization uint8
	vertQuantization uint8
	align Align

	scale fract.Unit
	logicalSize fract.Unit
	scaledSize fract.Unit
	
	blendMode BlendMode
	fontIndex uint8
	_ uint8
	_ uint8
}

// Creates a new [Renderer].
//
// Setting a font through [Renderer.SetFont]() or [Renderer.SetFontBytes]()
// is required before being able to operate with it. It's also heavily
// recommended to set a cache (none by default) right from the start.
func NewRenderer() *Renderer {
	renderer := &Renderer{}
	renderer.initBasicProps()
	return renderer
}

// To be chained with initBasicProps().
func (self *Renderer) missingBasicProps() bool {
	return (self.internalFlags & internalFlagBasicProps) == 0
}

// Virtually always preceded by if self.missingBasicProps().
func (self *Renderer) initBasicProps() {
	self.fontColor = color.RGBA{255, 255, 255, 255}
	self.horzQuantization = uint8(QtFull)
	self.vertQuantization = uint8(QtFull)
	self.align = Left | Baseline
	self.scale = 64
	self.logicalSize = 16*64
	self.scaledSize  = 16*64
	self.internalFlags |= internalFlagBasicProps
	if self.cacheHandler == nil {
		self.cacheHandler = pkgNoCacheHandler
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

// Sets the font to be used on subsequent operations. Without a
// font, a renderer is fundamentally useless, so don't forget to
// set this up.
//
// See also [Renderer.SetFontBytes]().
func (self *Renderer) SetFont(font *sfnt.Font) {
	// Notice: you *can* call this function with a nil font, but
	//         only if you *really really have to ensure* that the
	//         font can be released by the garbage collector while
	//         this renderer still exists... which is almost never.
	fontIndex := int(self.fontIndex)

	// skip if trying to assign a nil font beyond current slice bounds
	if font == nil && len(self.fonts) <= fontIndex { return }

	// ensure there's enough space in the fonts slice
	self.fonts = ensureSliceSize(self.fonts, fontIndex + 1)

	// assign font if new
	if font == self.fonts[fontIndex] { return }
	self.fonts[fontIndex] = font
	
	// notify font change
	self.notifyFontChange(font)
}

func (self *Renderer) notifyFontChange(font *sfnt.Font) {
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFontChange(font)
	}
	if self.fontSizer != nil {
		self.fontSizer.NotifyChange(font, &self.Buffer, self.scaledSize)
	}
}


// Utility method to set the font by passing its raw data and letting
// the renderer parse it. This method should be avoided if you want
// to reuse the font data at different points in your application; in
// that case, parsing the font only once (e.g. using [font.Library])
// and setting it with [Renderer.SetFont]() is the way to go.
func (self *Renderer) SetFontBytes(data []byte) error {
	font, err := sfnt.Parse(data)
	if err != nil { return err }
	self.SetFont(font)
	return nil
}

// Returns the current font. The font is nil by default.
func (self *Renderer) GetFont() *sfnt.Font {
	id := int(self.fontIndex)
	if len(self.fonts) <= id { return nil }
	return self.fonts[id]
}

// Sets the blend mode to be used on subsequent operations.
// The default blend mode will compose glyphs over the active
// target with regular alpha blending.
func (self *Renderer) SetBlendMode(blendMode BlendMode) {
	self.blendMode = blendMode
}

// Returns the renderer's [BlendMode]. As far as I know, this is only
// strictly necessary when implementing draw operations with custom
// shaders.
func (self *Renderer) GetBlendMode() BlendMode {
	return self.blendMode
}

// Sets the color to be used on subsequent draw operations.
// The default color is white.
func (self *Renderer) SetColor(fontColor color.Color) {
	if self.missingBasicProps() { self.initBasicProps() }
	self.fontColor = fontColor
}

// Returns the current drawing color.
func (self *Renderer) GetColor() color.Color {
	if self.missingBasicProps() { self.initBasicProps() }
	return self.fontColor
}

// Returns the current glyph cache handler, which is nil by default.
//
// Rarely used unless you are examining the cache handler manually.
func (self *Renderer) GetCacheHandler() cache.GlyphCacheHandler {
	if self.cacheHandler == pkgNoCacheHandler { return nil }
	return self.cacheHandler
}

// See the following [Renderer.SetCache8MiB]() method.
var pkgCache8MiB *cache.DefaultCache

// Utility method to set a cache that will get you started.
// For a more manual and adjustable approach, see
// [Renderer.SetCacheHandler]().
func (self *Renderer) SetCache8MiB() {
	// This uses a package level cache that will be shared by
	// all renderers using this utility method. If you have
	// many renderers with different fonts, it may be better
	// to start creating your own caches.
	if pkgCache8MiB == nil {
		pkgCache8MiB = cache.NewDefaultCache(8*1024*1024)
	}
	self.SetCacheHandler(pkgCache8MiB.NewHandler())
}

// Sets the glyph cache handler used by the renderer. By default,
// no cache is used, but you almost always want to set one.
//
// The easiest way is to use [Renderer.SetCache8MiB](), but that's
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
		if !self.missingBasicProps() { self.cacheHandler = pkgNoCacheHandler }
		if self.rasterizer != nil { self.rasterizer.SetOnChangeFunc(nil) }
		return
	}

	if self.rasterizer != nil {
		self.rasterizer.SetOnChangeFunc(cacheHandler.NotifyRasterizerChange)
	}

	cacheHandler.NotifySizeChange(self.scaledSize)
	font := self.GetFont()
	if font != nil { cacheHandler.NotifyFontChange(font) }
	if self.rasterizer != nil {
		cacheHandler.NotifyRasterizerChange(self.rasterizer)
	}
}
