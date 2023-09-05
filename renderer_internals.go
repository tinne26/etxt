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
	return logicalSize.MulDown(self.state.scale) // *
	// * I prefer MulDown to compensate having used FromFloat64Up()
	//   on both size and scale conversions. It's not a big deal in
	//   either case, but this reduces the maximum potential error.
}

// loadGlyphMask loads the mask for the given glyph at the given fractional
// pixel position. The renderer's cache handler, font, size, rasterizer and
// mask format are all taken into account.
// Precondition: !self.missingBasicProps(), rasterizer initialized,
// origin position communicated to the cache if relevant.
func (self *Renderer) loadGlyphMask(font *sfnt.Font, index sfnt.GlyphIndex, origin fract.Point) GlyphMask {
	// if the mask is available in the cache, that's all
	if self.cacheHandler != nil {
		glyphMask, found := self.cacheHandler.GetMask(index)
		if found { return glyphMask }
	}

	// glyph mask not cached, let's rasterize on our own
	segments, err := font.LoadGlyph(&self.buffer, index, fixed.Int26_6(self.state.scaledSize), nil)
	if err != nil {
		// if you need to deal with missing glyphs, you should do so before
		// reaching this point with functions like GetMissingRunes() and
		// replacing the relevant runes or glyphs
		panic("font.LoadGlyph(index = " + strconv.Itoa(int(index)) + ") error: " + err.Error())
	}

	// rasterize the glyph mask
	alphaMask, err := mask.Rasterize(segments, self.state.rasterizer, origin)
	if err != nil { panic("RasterizeGlyphMask failed: " + err.Error()) }

	// pass to cache and return
	glyphMask := convertAlphaImageToGlyphMask(alphaMask)
	if self.cacheHandler != nil {
		self.cacheHandler.PassMask(index, glyphMask)
	}
	return glyphMask
}

// --- internal functions for draw and renderer ---
// Precondition: sizer and font have been validated to be initialized.

func (self *Renderer) getOpKernBetween(prevGlyphIndex, currGlyphIndex sfnt.GlyphIndex) fract.Unit {
	return self.state.fontSizer.Kern(
		self.state.activeFont, &self.buffer, self.state.scaledSize,
		prevGlyphIndex, currGlyphIndex,
	)
}

func (self *Renderer) getOpAdvance(currGlyphIndex sfnt.GlyphIndex) fract.Unit {
	return self.state.fontSizer.GlyphAdvance(
		self.state.activeFont, &self.buffer, self.state.scaledSize, currGlyphIndex,
	)
}

func (self *Renderer) getOpLineAdvance(lineBreakNth int) fract.Unit {
	return self.state.fontSizer.LineAdvance(
		self.state.activeFont, &self.buffer, self.state.scaledSize, lineBreakNth,
	)
}

func (self *Renderer) getOpLineHeight() fract.Unit {
	return self.state.fontSizer.LineHeight(
		self.state.activeFont, &self.buffer, self.state.scaledSize,
	)
}

func (self *Renderer) getOpAscent() fract.Unit {
	return self.state.fontSizer.Ascent(
		self.state.activeFont, &self.buffer, self.state.scaledSize,
	)
}

func (self *Renderer) getOpDescent() fract.Unit {
	return self.state.fontSizer.Descent(
		self.state.activeFont, &self.buffer, self.state.scaledSize,
	)
}

// Notice: this is rather slow, uncached. I'm leaving it like this because
// it's rarely used anyway, and in the grand scheme of things, when this is
// actually required, most of the runtime will still go to actual font
// rendering... but *there is* room for improvement (especially if we were
// not using golang's sfnt package, which is kinda shitty).
func (self *Renderer) getSlowOpXHeight() fract.Unit {
	const hintingNone = 0
	metrics, err := self.state.activeFont.Metrics(&self.buffer, fixed.Int26_6(self.state.scaledSize), hintingNone)
	if err != nil { panic("font.Metrics error: " + err.Error()) }
	return fract.Unit(metrics.XHeight)
}
