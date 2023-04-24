package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to perform low level operations related to
// raw font glyphs and glyph masks.
//
// In general, this type is used through method chaining:
//   renderer.Glyph().LoadMask(glyphIndex, dot)
//
// This type also uses fractional units for many operations, so it's
// advisable to be familiar with [RendererFract] and the [etxt/fract]
// subpackage before diving in.
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
type RendererGlyph Renderer

// [Gateway] to [RendererGlyph] functionality.
//
// [Gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
func (self *Renderer) Glyph() *RendererGlyph {
	return (*RendererGlyph)(self)
}

// Default draw glyph function.
func (self *RendererGlyph) DrawMask(mask GlyphMask, dot fract.Point) {
	panic("unimplemented")
}

// Load a glyph mask. Most often used for manual caching.
func (self *RendererGlyph) LoadMask(index sfnt.GlyphIndex, dot fract.Point) GlyphMask {
	return (*Renderer)(self).glyphLoadMask(index, dot)
}

func (self *RendererGlyph) RuneIndex(codePoint rune) sfnt.GlyphIndex {
	return (*Renderer)(self).glyphRuneIndex(codePoint)
}

// Sets the glyph mask rasterizer to be used on subsequent operations.
// Nil rasterizers are not allowed.
func (self *RendererGlyph) SetRasterizer(rasterizer mask.Rasterizer) {
	(*Renderer)(self).glyphSetRasterizer(rasterizer)
}

// Returns the current glyph mask rasterizer.
//
// This function is only useful when working with configurable rasterizers;
// ignore it if you are using the default glyph mask rasterizer.
//
// Mask rasterizers are not concurrent-safe, so be careful with
// what you do and where you put them.
func (self *RendererGlyph) GetRasterizer() mask.Rasterizer {
	return (*Renderer)(self).glyphGetRasterizer()
}

// ---- underlying implementations ----

func (self *Renderer) glyphLoadMask(index sfnt.GlyphIndex, dot fract.Point) GlyphMask {
	if self.missingBasicProps() { self.initBasicProps() }
	self.initRasterizer()
	self.cacheHandler.NotifyFractChange(dot)
	return self.loadGlyphMask(self.GetFont(), index, dot)
}

func (self *Renderer) glyphRuneIndex(codePoint rune) sfnt.GlyphIndex {
	return self.getGlyphIndex(self.GetFont(), codePoint)
}

func (self *Renderer) glyphGetRasterizer() mask.Rasterizer {
	self.initRasterizer()
	return self.rasterizer
}

func (self *Renderer) initRasterizer() {
	if self.internalFlags & internalFlagRasterizer == 0 {
		self.internalFlags |= internalFlagRasterizer
		self.rasterizer = &mask.DefaultRasterizer{}
	}
}

func (self *Renderer) glyphSetRasterizer(rasterizer mask.Rasterizer) {
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
