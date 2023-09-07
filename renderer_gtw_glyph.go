package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to perform low level operations related to
// raw font glyphs and rasterizers.
//
// In general, this type is used through method chaining:
//   renderer.Glyph().LoadMask(glyphIndex, origin)
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

// Default glyph drawing function. This is a very low level function,
// almost only relevant if you are trying to implement custom draw
// functions for [RendererGlyph.SetDrawFunc]().
func (self *RendererGlyph) DrawMask(target Target, mask GlyphMask, origin fract.Point) {
	(*Renderer)(self).glyphDrawMask(target, mask, origin)
}

// Loads a glyph mask. This is a very low level function, almost only
// relevant if you are trying to implement custom draw functions for
// [RendererGlyph.SetDrawFunc]().
func (self *RendererGlyph) LoadMask(index sfnt.GlyphIndex, origin fract.Point) GlyphMask {
	return (*Renderer)(self).glyphLoadMask(index, origin)
}

// Overrides the renderer's glyph drawing function with a custom
// one. You can set it to nil to go back to the default behavior.
//
// The default implementation is a streamlined equivalent of:
//   mask := renderer.Glyph().LoadMask(glyphIndex, origin)
//   renderer.Glyph().DrawMask(target, mask, origin)
// See [examples/ebiten/colorful] and [examples/ebiten/shaking]
// for further customization examples.
//
// Changing renderer properties dynamically on the glyph drawing
// function is generally unsafe and many behaviors are undefined.
// Only the text color can be safely changed at the moment.
//
// [examples/ebiten/colorful]: https://github.com/tinne26/etxt/blob/main/examples/ebiten/colorful/main.go
// [examples/ebiten/shaking]: https://github.com/tinne26/etxt/blob/main/examples/ebiten/shaking/main.go
func (self *RendererGlyph) SetDrawFunc(drawFn func(Target, sfnt.GlyphIndex, fract.Point)) {
	// Note: we could actually allow the font and the sizer to be changed, but this
	//       has implications on measuring. We wouldn't be able to measure properly,
	//       which also has impact on centering and other relative positioning
	//       operations. So, it's complicated.
	self.customDrawFn = drawFn
}

// Obtains the glyph index for the given rune in the current renderer's
// font. Panics if the glyph index can't be found.
//
// If you need to know whether the glyph mapping exists or not, consider
// [font.GetMissingRunes]() instead... or the manual approach:
//   buffer := renderer.GetBuffer()
//   font := renderer.GetFont()
//   index, err := font.GlyphIndex(buffer, codePoint)
//   if err != nil { /* handle */ }
//   if index == 0 { /* handle notdef glyph */ }
//
// [font.GetMissingRunes]: https://pkg.go.dev/github.com/tinne26/etxt/font#GetMissingRunes
func (self *RendererGlyph) GetRuneIndex(codePoint rune) sfnt.GlyphIndex {
	return (*Renderer)(self).glyphRuneIndex(codePoint)
}

// Caches the given glyph with the current font and scaled size.
// The caching is attempted for each fractional position allowed
// by the current quantization configuration.
//
// Notice that the success of this method depends on the renderer's
// cache configuration too. If there's no cache, the cache doesn't
// have enough capacity or you are using a custom cache with an
// unusual caching policy, results may not be what you expect.
func (self *RendererGlyph) CacheIndex(index sfnt.GlyphIndex) {
	(*Renderer)(self).glyphCacheIndex(index)
}

// Sets the glyph mask rasterizer to be used on subsequent operations.
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

func (self *Renderer) glyphLoadMask(index sfnt.GlyphIndex, origin fract.Point) GlyphMask {
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(origin)
	}
	return self.loadGlyphMask(self.state.activeFont, index, origin)
}

func (self *Renderer) glyphDrawMask(target Target, mask GlyphMask, origin fract.Point) {
	if self.cacheHandler != nil {
		self.cacheHandler.NotifyFractChange(origin)
	}
	self.defaultDrawFunc(target, origin, mask)
}

func (self *Renderer) glyphRuneIndex(codePoint rune) sfnt.GlyphIndex {
	return self.getGlyphIndex(self.state.activeFont, codePoint)
}

func (self *Renderer) glyphCacheIndex(index sfnt.GlyphIndex) {
	horzQuant, vertQuant := self.fractGetQuantization()
	for y := fract.Unit(0); y < fract.One; y += vertQuant {
		for x := fract.Unit(0); x < fract.One; x += horzQuant {
			origin := fract.UnitsToPoint(x, y)
			_ = self.glyphLoadMask(index, origin)
		}
	}
}

func (self *Renderer) glyphGetRasterizer() mask.Rasterizer {
	return self.state.rasterizer
}

func (self *Renderer) glyphSetRasterizer(rasterizer mask.Rasterizer) {
	// clear existing rasterizer onChangeFunc
	if self.state.rasterizer != nil {
		self.state.rasterizer.SetOnChangeFunc(nil)
	}

	// set new rasterizer and link it to the cache handler
	self.state.rasterizer = rasterizer
	if self.cacheHandler == nil {
		rasterizer.SetOnChangeFunc(nil)
	} else {
		rasterizer.SetOnChangeFunc(self.cacheHandler.NotifyRasterizerChange)
		self.cacheHandler.NotifyRasterizerChange(rasterizer)
	}
}
