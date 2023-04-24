//go:build nope
package etxt

// TODO: once Text() mod is added, maybe I want to leave this a single Glyph()
//       and put the low level LoadMask and stuff.

// An alternative interface for using a [Renderer] directly with glyphs
// instead of strings.
//
// None of the functions will create or store references to the glyph slices
// passed as input arguments, so it's safe to reuse the memory for subsequent
// operations.
type RendererGlyphs Renderer

func (self *Renderer) Glyphs() *RendererGlyphs {
	return (*RendererModGlyph)(self)
}

// No references to the 'text' slice will be kept once the function ends, so it's
// safe to reuse the slice memory for subsequent operations.
func (self *RendererGlyphs) Measure(text []sfnt.GlyphIndex) fract.Rect {
	
}

// Default draw glyph function.
func (self *RendererGlyphs) DrawMaskFn(dot fract.Point, mask GlyphMask, glyph sfnt.GlyphIndex) {}

// TODO: this may be important for pre-caching.
func (self *RendererGlyphs) LoadMask(index sfnt.GlyphIndex, dot fract.Point) GlyphMask {
	return (*Renderer)(self).loadMask(index, dot)
}

// No references to the 'text' slice will be kept once the function ends, so it's
// safe to reuse the slice memory for subsequent operations.
func (self *RendererGlyphs) Draw(target TargetImage, text []sfnt.GlyphIndex, x, y fract.Unit) fract.Point {
	
}

// No references to the 'text' slice will be kept once the function ends, so it's
// safe to reuse the slice memory for subsequent operations.
// TODO: what's the point of origin, really? We still don't know direction or
//       others, so it all sounds a bit dangerous
// func (self *RendererGlyphs) Origin(text []sfnt.GlyphIndex, x, y fract.Unit) (fract.Unit, fract.Unit) {
//
// }

// No references to the 'text' slice will be kept once the function ends, so it's
// safe to reuse the slice memory for subsequent operations.
// func (self *RendererGlyphs) Traverse(text []sfnt.GlyphIndex, x, y fract.Unit, fn) (nextX, nextY fract.Unit) {
//	
// }

// ---- underlying implementations ----

func (self *Renderer) loadMask(index sfnt.GlyphIndex, dot fract.Point) GlyphMask {
	return self.loadGlyphMask(self.GetFont(), index, dot)
}
