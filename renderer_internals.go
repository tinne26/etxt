package etxt

import "strconv"

import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"

func (self *Renderer) getGlyphIndex(font *sfnt.Font, codePoint rune) sfnt.GlyphIndex {
	index, err := font.GlyphIndex(&self.buffer, codePoint)
	if err != nil { panic("font.GlyphIndex error: " + err.Error()) }
	if index == 0 {
		msg := "glyph index for '" + string(codePoint) + "' ["
		msg += runeToUnicodeCode(codePoint) + "] missing"
		panic(msg)
	}
	return index
}

func (self *Renderer) scaleLogicalSize(logicalSize fract.Unit) fract.Unit {
	return logicalSize.MulDown(self.scale) // *
	// * I prefer MulDown to compensate having used FromFloat64Up()
	//   on both size and scale conversions. It's not a big deal in
	//   either case, but this reduces the maximum potential error.
}

// loadGlyphMask loads the mask for the given glyph at the given fractional
// pixel position. The renderer's cache handler, font, size, rasterizer and
// mask format are all taken into account.
// Precondition: !self.missingBasicProps()
func (self *Renderer) loadGlyphMask(font *sfnt.Font, index sfnt.GlyphIndex, dot fract.Point) GlyphMask {
	// if the mask is available in the cache, that's all
	glyphMask, found := self.cacheHandler.GetMask(index)
	if found { return glyphMask }

	// glyph mask not cached, let's rasterize on our own
	segments, err := font.LoadGlyph(&self.buffer, index, fixed.Int26_6(self.scaledSize), nil)
	if err != nil {
		// if you need to deal with missing glyphs, you should do so before
		// reaching this point with functions like GetMissingRunes() and
		// replacing the relevant runes or glyphs
		panic("font.LoadGlyph(index = " + strconv.Itoa(int(index)) + ") error: " + err.Error())
	}

	// rasterize the glyph mask
	alphaMask, err := mask.Rasterize(segments, self.rasterizer, dot)
	if err != nil { panic("RasterizeGlyphMask failed: " + err.Error()) }

	// pass to cache and return
	glyphMask = convertAlphaImageToGlyphMask(alphaMask)
	self.cacheHandler.PassMask(index, glyphMask)
	return glyphMask
}
