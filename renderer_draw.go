package etxt

import "strconv"

import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt/emask"

// Drawing functions for the Renderer type.

// Notice:
//   If golang supported union types, we could have a single function
//   accepting strings, runes, bytes, glyph indices... and also both integer
//   or fractional coordinates as fixed.Int26_6. Since that's not happening
//   yet, I decided to heavily cut the API surface area and leave advanced
//   usage to Traverse* and other functions, instead of spamming with all the
//   Draw(Fract)?(Each)?(Glyphs)? variants.

// Draw the given text with the current configuration (font, size, color,
// target, etc). The position at which the text will be drawn depends on
// the given pixel coordinates and the renderer's align (see SetAlign rules).
//
// The returned value should be ignored except on advanced use-cases
// (refer to Traverse's documentation).
//
// Missing glyphs in the current font will cause the renderer to panic.
// See GetMissingRunes if you need to make your system more robust.
//
// Line breaks encoded as \n will be handled automatically.
func (self *Renderer) Draw(text string, x, y int) fixed.Point26_6 {
	fx, fy := fixed.Int26_6(x << 6), fixed.Int26_6(y << 6)
	return self.DrawFract(text, fx, fy)
}

// Exactly the same as Draw, but accepting [fractional pixel] coordinates.
//
// Notice that passing a fractional coordinate won't make the draw operation
// be fractionally aligned by itself, that still depends on the renderer's
// quantization mode.
//
// [fractional pixel] https://github.com/tinne26/etxt/docs/fixed-26-6.md
func (self *Renderer) DrawFract(text string, x, y fixed.Int26_6) fixed.Point26_6 {
	// safety checks
	if self.target == nil { panic("draw called while target == nil (tip: renderer.SetTarget())") }
	if self.font   == nil { panic("draw called while font == nil (tip: renderer.SetFont())"    ) }
	if text == "" { return fixed.Point26_6{ X: x, Y: y } }

	// traverse text and draw each glyph
	return self.Traverse(text, fixed.Point26_6{ X: x, Y: y },
		func(currentDot fixed.Point26_6, codePoint rune, glyphIndex GlyphIndex) {
			if codePoint == '\n' { return }
			mask := self.LoadGlyphMask(glyphIndex, currentDot)
			self.DefaultDrawFunc(currentDot, mask, glyphIndex)
		})
}

// Low-level function used with Traverse* or Feed* operations.
//
// LoadGlyphMask loads the mask for the given glyph at the given fractional
// pixel position. The renderer's cache handler, font, size, rasterizer and
// mask format are all taken into account.
func (self *Renderer) LoadGlyphMask(index GlyphIndex, dot fixed.Point26_6) GlyphMask {
	// if the mask is available in the cache, that's all
	if self.cacheHandler != nil {
		glyphMask, found := self.cacheHandler.GetMask(index)
		if found { return glyphMask }
	}

	// glyph mask not cached, let's rasterize on our own
	segments, err := self.font.LoadGlyph(&self.buffer, index, self.sizePx, nil)
	if err != nil {
		// if you need to deal with missing glyphs, you should do so before
		// reaching this point with functions like GetMissingRunes() and
		// replacing the relevant runes or glyphs
		panic("font.LoadGlyph(index = " + strconv.Itoa(int(index)) + ") error: " + err.Error())
	}

	// rasterize the glyph mask
	alphaMask, err := emask.Rasterize(segments, self.rasterizer, dot)
	if err != nil { panic("RasterizeGlyphMask failed: " + err.Error()) }

	// pass to cache and return
	glyphMask := convertAlphaImageToGlyphMask(alphaMask)
	if self.cacheHandler != nil {
		self.cacheHandler.PassMask(index, glyphMask)
	}
	return glyphMask
}
