package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to [Twine] operations and related configurations.
//
// In general, this type is used through method chaining:
//   renderer.Twine().Draw(canvas, twine, x, y)
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.6#Renderer
type RendererTwine Renderer

// [Gateway] to [RendererTwine] functionality.
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.6#Renderer
func (self *Renderer) Twine() *RendererTwine {
	return (*RendererTwine)(self)
}

// Registers the given font at the specified index. Having multiple fonts
// is useful when trying to combine regular, bold and italic subfamilies
// in the same draw call, changing fonts on the fly and so on.
//
// You may use [NextFontIndex] to register the font at the next available
// index, which you will get back as the return value. Otherwise, the index
// value must be in [0..255] (inclusive) and the return value can be
// ignored, as it will be the same as the given index.
//
// To change the active font, use [RendererTwine.SetFontIndex]().
//
// Registering a font on the active font index is safe and equivalent
// to the more basic [Renderer.SetFont]().
func (self *RendererTwine) RegisterFont(index FontIndex, font *sfnt.Font) FontIndex {
	// Additional technical details:
	// - There is no way to release the underlying font slice except letting
	//   the whole renderer be garbage collected.
	// - Setting a nil font to an index beyond the bounds of the underlying
	// 	  slice won't panic or be ignored, it will make the slice grow.
	return (*Renderer)(self).twineRegisterFont(index, font)
}

// NOTE: if you need this exposed, explain the reason and let me know.
//       At the moment, I think I'd rather expose an EachFont() iteration
//       method.
// Returns the renderer fonts available for use with twines.
// This method is exposed mostly for completeness, and you should
// ignore it / avoid it in most cases.
// 
// The returned slice should be treated as read only. At the very
// least, know that modifying the active font externally will
// leave it unsynced with the renderer's cache handler and sizer.
// func (self *RendererTwine) GetFonts() []*sfnt.Font {
// 	return self.fonts
// }

// Sets the active font index to the given value.
//
// For more context, see [FontIndex] and [RendererTwine.RegisterFont]().
func (self *RendererTwine) SetFontIndex(index FontIndex) {
	// technical note: if the index exceeds the bounds of the
	// underlying slice, the slice will be resized to make the
	// index referenceable.
	(*Renderer)(self).twineSetFontIndex(index)
}

// Returns the index of the renderer's active font.
func (self *RendererTwine) GetFontIndex() FontIndex {
	return self.state.fontIndex
}

// Same as [Renderer.Measure](), but expecting a [Twine] instead of a string.
func (self *RendererTwine) Measure(twine Twine) fract.Rect {
	return (*Renderer)(self).twineMeasure(twine)
}

// Same as [Renderer.Draw](), but expecting a [Twine] instead of a string.
//
// This method should be mostly functional at the moment, but it's still
// WIP. Current list of limitations that we might relax in the future:
//  - Text direction can't be changed in the middle of the text.
//  - No DrawWithWrap() version available.
// Other limitations, like quantization not being allowed to change
// while drawing, are expected to be permanent. Regarding size changes,
// some unique conditions apply, see [Twine.AddLineMetricsRefresh]().
func (self *RendererTwine) Draw(target Target, twine Twine, x, y int) {
	(*Renderer)(self).twineDraw(target, twine, x, y)
}

// Registers a custom callback that can be triggered for specific text
// fragments or positions while drawing a [Twine]. See [TwineEffectFunc] for
// additional details.
//
// You may use [NextEffectKey] to register the function at the next available
// index, which you will get back as the return value. Otherwise, the key
// value must be in [0..192] (inclusive) and the return value can be ignored,
// as it will be the same as the given key.
func (self *RendererTwine) RegisterEffectFunc(key TwineEffectKey, fn TwineEffectFunc) TwineEffectKey {
	// Additional technical docs:
	// - If the index exceeds the bounds of the underlying slice, the slice will be
	//   resized to make the index referenceable. 
	// - Unless you let the whole renderer be garbage collected, there is no way to
	//   release the underlying slice.
	return (*Renderer)(self).twineRegisterEffectFunc(key, fn)
}

// Registers a motion callback that can be triggered for each glyph within
// a designated [Twine] fragment. See [TwineMotionFunc] for more details.
//
// You may use [NextEffectKey] to register the function at the next available
// index, which you will get back as the return value. Otherwise, the key
// value must be in [0..192] (inclusive) and the return value can be ignored,
// as it will be the same as the given key.
func (self *RendererTwine) RegisterMotionFunc(key TwineMotionKey, fn TwineMotionFunc) TwineMotionKey {
	return (*Renderer)(self).twineRegisterMotionFunc(key, fn)
}

// NOTE: if you need this exposed, explain the reason and let me know.
// Returns the renderer's underlying slice of registered [TwineEffectFunc] functions.
// See also [RendererTwine.RegisterEffectFunc](). Operate at your own risk.
// func (self *RendererTwine) GetEffectFuncs() []TwineEffectFunc {
// 	return self.twineEffectFuncs
// }

// NOTE: if you need this exposed, explain the reason and let me know.
// Returns the renderer's underlying slice of registered [TwineMotionFunc] functions.
// See also [RendererTwine.RegisterMotionFunc](). Operate at your own risk.
// func (self *RendererTwine) GetMotionFuncs() []TwineMotionFunc {
// 	return self.twineMotionFuncs
// }

// ---- implementations ----

func (self *Renderer) twineRegisterFont(index FontIndex, font *sfnt.Font) FontIndex {
	if index == 255 {
		iindex := len(self.fonts)
		if iindex >= 255 { panic("can't register more than 254 fonts") }
		index = FontIndex(iindex)
	}

	if index == self.state.fontIndex {
		self.SetFont(font)
	} else {
		self.fonts = ensureSliceSize(self.fonts, int(index + 1))
		self.fonts[index] = font
	}

	return index
}

func (self *Renderer) twineSetFontIndex(index FontIndex) {
	// grow the slice if necessary (as per spec)
	self.fonts = ensureSliceSize(self.fonts, int(index) + 1)

	// change active font index and notify change if relevant
	self.state.fontIndex = index
	newFont := self.fonts[index]
	if newFont != self.state.activeFont {
		self.state.activeFont = newFont
		self.notifyFontChange(newFont)
	}
}

func (self *Renderer) twineRegisterEffectFunc(key TwineEffectKey, fn TwineEffectFunc) TwineEffectKey {
	if key > 192 {
		if key == 255 {
			ikey := len(self.twineEffectFuncs)
			if ikey >= 193 { panic("can't register more than 192 TwineEffectFuncs") }
			key = TwineEffectKey(ikey)
		} else {
			panic("TextFuncIndices above 192 are reserved for internal use")
		}
	}
	
	self.twineEffectFuncs = ensureSliceSize(self.twineEffectFuncs, int(key) + 1)
	self.twineEffectFuncs[key] = fn
	return key
}

func (self *Renderer) twineRegisterMotionFunc(key TwineMotionKey, fn TwineMotionFunc) TwineMotionKey {
	if key > 192 {
		if key == 255 {
			ikey := len(self.twineMotionFuncs)
			if ikey >= 193 { panic("can't register more than 192 TwineMotionFuncs") }
			key = TwineMotionKey(ikey)
		} else {
			panic("TextFuncIndices above 192 are reserved for internal use")
		}
	}
	
	self.twineMotionFuncs = ensureSliceSize(self.twineMotionFuncs, int(key) + 1)
	self.twineMotionFuncs[key] = fn
	return key
}
