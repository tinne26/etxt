package etxt

import "strconv"
import "image/color"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/efixed"
import "github.com/tinne26/etxt/emask"
import "github.com/tinne26/etxt/ecache"
import "github.com/tinne26/etxt/esizer"

// This file contains the Renderer type definition and all the
// getter and setter methods. Actual operations are split in other
// files.

// The Renderer is the main type for drawing text provided by etxt.
//
// Renderers allow you to control font, text size, color, text
// alignment and more from a single place.
//
// Basic usage goes like this:
//  - Create and store a renderer.
//  - Adjust its properties as desired, call Draw as many times
//    as you need... and keep repeating.
//
// If you need more advice or guidance, check the [renderers document]
// and the [examples].
//
// [renderers document]: https://github.com/tinne26/etxt/blob/main/docs/renderer.md
// [examples]: https://github.com/tinne26/etxt/tree/main/examples
type Renderer struct {
	font *Font
	target TargetImage
	mainColor color.Color
	sizer esizer.Sizer

	vertAlign VertAlign
	horzAlign HorzAlign
	direction Direction

	quantization QuantizationMode
	mixMode MixMode

	sizePx fixed.Int26_6
	lineSpacing fixed.Int26_6 // 64 by default (which is 1 in 26_6)
	lineHeight  fixed.Int26_6 // non-negative value or -1 to match font height
	cachedLineAdvance fixed.Int26_6
	lineAdvanceIsCached bool

	cacheHandler ecache.GlyphCacheHandler
	rasterizer emask.Rasterizer
	metrics *font.Metrics // cached for the current font and size
	buffer sfnt.Buffer
}

// Creates a new renderer with the default vector rasterizer.
// See NewRenderer()'s documentation for more details.
func NewStdRenderer() *Renderer {
	return NewRenderer(&emask.DefaultRasterizer{})
}

// Creates a new Renderer with the given glyph mask rasterizer.
// For the default rasterizer, see NewStdRenderer() instead.
//
// After creating a renderer, you must set at least the font and
// the target in order to be able to draw. In most cases, you will
// also want to set a cache handler and a color. Check the setter
// functions for more details on all those.
//
// Renderers are not safe for concurrent use.
func NewRenderer(rasterizer emask.Rasterizer) *Renderer {
	return &Renderer{
		sizer: &esizer.DefaultSizer{},
		vertAlign: Baseline,
		horzAlign: Left,
		direction: LeftToRight,
		lineSpacing: fixed.Int26_6(1 << 6),
		lineHeight: -1,
		sizePx: 16 << 6,
		quantization: QuantizeFull,
		rasterizer: rasterizer,
		mainColor: color.RGBA{ 255, 255, 255, 255 },
		mixMode: defaultMixMode,
	}
}

// Sets the font to be used on subsequent operations.
// Make sure to set one before starting to draw!
func (self *Renderer) SetFont(font *Font) {
	// Notice: you *can* call this function with a nil font, but
	//         only if you *really really have to ensure* that the
	//         font can be released by the garbage collector while
	//         this renderer still exists... which is almost never.
	if font == self.font { return }
	self.font = font

	// drop cached information
	self.lineAdvanceIsCached = false
	self.metrics = nil
}

// Gets the current font. The font is nil by default.
func (self *Renderer) GetFont() *Font { return self.font }

// Sets the target of subsequent operations.
// Attempting to draw without a target will cause the
// renderer to panic.
//
// You can also clear the target by setting it to nil once
// it's no longer needed.
func (self *Renderer) SetTarget(target TargetImage) {
	self.target = target
}

// Sets the mix mode to be used on subsequent operations.
// The default mix mode will compose glyphs over the active
// target with regular alpha blending.
func (self *Renderer) SetMixMode(mixMode MixMode) {
	self.mixMode = mixMode
}

// Sets the color to be used on subsequent operations.
// The default color is white.
func (self *Renderer) SetColor(mainColor color.Color) {
	self.mainColor = mainColor
}

// Returns the current font color.
func (self *Renderer) GetColor() color.Color { return self.mainColor }

// Sets the quantization mode to be used on subsequent operations.
// By default, the renderer's mode is QuantizeFull.
func (self *Renderer) SetQuantizationMode(mode QuantizationMode) {
	self.quantization = mode
}

// Gets the current glyph cache handler, which is nil by default.
//
// Rarely used unless you are examining the cache handler manually.
func (self *Renderer) GetCacheHandler() ecache.GlyphCacheHandler {
	return self.cacheHandler
}

// Sets the glyph cache handler used by the renderer. By default,
// no cache is used, but you almost always want to set one, e.g.:
//   cache := etxt.NewDefaultCache(16*1024*1024) // 16MB
//   textRenderer.SetCacheHandler(cache.NewHandler())
func (self *Renderer) SetCacheHandler(cacheHandler ecache.GlyphCacheHandler) {
	self.cacheHandler = cacheHandler
	if self.rasterizer != nil {
		self.rasterizer.SetOnChangeFunc(cacheHandler.NotifyRasterizerChange)
	}

	if cacheHandler == nil { return }
	cacheHandler.NotifySizeChange(self.sizePx)
	if self.font != nil { cacheHandler.NotifyFontChange(self.font) }
	if self.rasterizer != nil {
		cacheHandler.NotifyRasterizerChange(self.rasterizer)
	}
}

// Gets the current glyph mask rasterizer.
//
// This function is only useful when working with configurable rasterizers;
// ignore it if you are using the default glyph mask rasterizer.
//
// Mask rasterizers are not concurrent-safe, so be careful with
// what you do and where you put them.
func (self *Renderer) GetRasterizer() emask.Rasterizer {
	return self.rasterizer
}

// Sets the glyph mask rasterizer to be used on subsequent operations.
func (self *Renderer) SetRasterizer(rasterizer emask.Rasterizer) {
	// clear rasterizer onChangeFunc
	if self.rasterizer != nil {
		self.rasterizer.SetOnChangeFunc(nil)
	}

	// set rasterizer
	self.rasterizer = rasterizer

	// link new rasterizer to the cache handler
	if rasterizer != nil {
		if self.cacheHandler == nil {
			rasterizer.SetOnChangeFunc(nil)
		} else {
			rasterizer.SetOnChangeFunc(self.cacheHandler.NotifyRasterizerChange)
			self.cacheHandler.NotifyRasterizerChange(rasterizer)
		}
	}
}

// Sets the font size that will be used on subsequent operations.
//
// Sizes are given in pixels and must be >= 1.
// By default, the renderer will draw text at a size of 16px.
//
// The relationship between font size and the size of its glyphs
// is complicated and can vary a lot between fonts, but
// to provide a [general reference]:
//  - A capital latin letter is usually around 70% as tall as
//    the given size. E.g: at 16px, "A" will be 10-12px tall.
//  - A lowercase latin letter is usually around 48% as tall as
//    the given size. E.g: at 16px, "x" will be 7-9px tall.
//
// [general reference]: https://github.com/tinne26/etxt/blob/main/docs/px-size.md
func (self *Renderer) SetSizePx(sizePx int) {
	self.SetSizePxFract(fixed.Int26_6(sizePx << 6))
}

// Like SetSizePx, but accepting a float64 fractional pixel size.
// func (self *Renderer) SetSizePxFloat(sizePx float64) {
// 	self.SetSizePxFract(efixed.FromFloat64RoundToZero(sizePx))
// }

// Like SetSizePx, but accepting a fractional pixel size in the
// form of a [26.6 fixed point] integer.
//
// [26.6 fixed point]: https://github.com/tinne26/etxt/blob/main/docs/fixed-26-6.md
func (self *Renderer) SetSizePxFract(sizePx fixed.Int26_6) {
	if sizePx < 64 { panic("sizePx must be >= 1") }
	if sizePx == self.sizePx { return }

	// set new size and check it's in range
	// (we are artificially limiting sizes so glyphs don't take
   // more than ~1GB on ebiten or ~0.25GB as alpha images. Even
   // at those levels most computers will choke to death if they
	// try to render multiple characters, but I tried...)
	self.sizePx = sizePx
	if self.sizePx & ^fixed.Int26_6(0x000FFFFF) != 0 {
		panic("sizePx " + strconv.FormatFloat(float64(sizePx)/64, 'f', 2, 64) + " too big")
	}

	// notify update to the cacheHandler
	if self.cacheHandler != nil {
		self.cacheHandler.NotifySizeChange(sizePx)
	}

	// drop cached information
	self.lineAdvanceIsCached = false
	self.metrics = nil
}

// Returns the current font size as a [fixed.Int26_6].
//
// [fixed.Int26_6]: https://github.com/tinne26/etxt/blob/main/docs/fixed-26-6.md
func (self *Renderer) GetSizePxFract() fixed.Int26_6 {
	return self.sizePx
}

// Sets the line height to be used on subsequent operations.
//
// Line height is only used when line breaks are found in the input
// text to be processed. Notice that line spacing will also affect the
// space between lines of text (lineAdvance = lineHeight*lineSpacing).
//
// The units are pixels, not points, and only non-negative values
// are allowed. If you need negative line heights for some reason,
// use negative line spacing factors instead.
//
// By default, the line height is set to auto (see SetLineHeightAuto).
func (self *Renderer) SetLineHeight(heightPx float64) {
	if heightPx < 0 {
		panic("negative line height not allowed, use negative line spacing instead")
	}
	// See SetLineSpacing notes a couple methods below.
	self.lineHeight = efixed.FromFloat64RoundToZero(heightPx)
	self.lineAdvanceIsCached = false
}

// Sets the line height to automatically match the height of the
// active font and size. This is the default behavior for line
// height.
//
// For manual line height configuration, see SetLineHeight.
func (self *Renderer) SetLineHeightAuto() {
	self.lineHeight = -1
	self.lineAdvanceIsCached = false
}

// Sets the line spacing to be used on subsequent operations.
// By default, the line spacing factor is 1.0.
//
// Line spacing is only applied when line breaks are found in the
// input text to be processed.
//
// Notice that line spacing and line height are different things.
// See SetLineHeight for more details.
func (self *Renderer) SetLineSpacing(factor float64) {
	// Line spacing will be quantized to a multiple of 1/64.
	// Providing a float64 that already adjusts to that will
	// guarantee a conversion that always takes the "fast" path.
	// You shouldn't worry too much about this, though.
	self.lineSpacing = efixed.FromFloat64RoundToZero(factor)
	self.lineAdvanceIsCached = false
}

// Returns the result of lineHeight*lineSpacing. You rarely
// need this unless you are drawing lines one by one and setting
// their y coordinate manually.
//
// The result is always unquantized and cached.
func (self *Renderer) GetLineAdvance() fixed.Int26_6 {
	if self.lineAdvanceIsCached { return self.cachedLineAdvance }

	var newLineAdvance fixed.Int26_6
	if self.lineHeight == -1 { // auto mode (match font height)
		if self.metrics == nil { self.updateMetrics() }
		if self.lineSpacing == 64 { // fast common case
			newLineAdvance = self.metrics.Height
		} else {
			// TODO: fixed.Int26_6.Mul() implementation is biased (rounding up).
			//       See: https://go.dev/play/p/UCzCWSBPesH
			//       In our case, only one value can be negative (lineSpacing),
			//       see if the current approach is biased or determine what's
			//       the appropriate bias (maybe round down on negative)
			newLineAdvance = fixed.Int26_6((int64(self.metrics.Height)*int64(self.lineSpacing)) >> 6)
		}
	} else { // manual mode, use stored line height
		if self.lineSpacing == 64 { // fast case
			newLineAdvance = self.lineHeight
		} else {
			newLineAdvance = fixed.Int26_6((int64(self.lineHeight)*int64(self.lineSpacing)) >> 6)
		}
	}

	self.cachedLineAdvance = newLineAdvance
	self.lineAdvanceIsCached = true
	return newLineAdvance
}

// See documentation for SetAlign.
func (self *Renderer) SetVertAlign(vertAlign VertAlign) {
	if vertAlign < Top || vertAlign > Bottom { panic("bad VertAlign") }
	self.vertAlign = vertAlign
}

// See documentation for SetAlign.
func (self *Renderer) SetHorzAlign(horzAlign HorzAlign) {
	if horzAlign < Left || horzAlign > Right { panic("bad HorzAlign") }
	self.horzAlign = horzAlign
}

// Configures how Draw* coordinates will be interpreted. For example:
//  - If the alignment is set to (etxt.Top, etxt.Left), coordinates
//    passed to subsequent operations will be interpreted as the
//    top-left corner of the box in which the text has to be drawn.
//  - If the alignment is set to (etxt.YCenter, etxt.XCenter), coor-
//    dinates passed to subsequent operations will be interpreted
//    as the center of the box in which the text has to be drawn.
//
// See https://github.com/tinne26/etxt/blob/main/docs/img/gtxt_aligns.png
// for a visual explanation instead.
//
// By default, the renderer's alignment is (etxt.Baseline, etxt.Left).
func (self *Renderer) SetAlign(vertAlign VertAlign, horzAlign HorzAlign) {
	self.SetVertAlign(vertAlign)
	self.SetHorzAlign(horzAlign)
}

// Returns the current align. See SetAlign documentation for more
// details on text align.
func (self *Renderer) GetAlign() (VertAlign, HorzAlign) {
	return self.vertAlign, self.horzAlign
}

// Sets the text direction to be used on subsequent operations.
//
// By default, the direction is LeftToRight.
func (self *Renderer) SetDirection(dir Direction) {
	if dir != LeftToRight && dir != RightToLeft { panic("bad direction") }
	self.direction = dir
}

// Gets the current Sizer. You shouldn't worry about sizers unless
// you are making custom glyph mask rasterizers or want to disable
// kerning or adjust spacing in some other unusual way.
func (self *Renderer) GetSizer() esizer.Sizer {
	return self.sizer
}

// Sets the current sizer, which must be non-nil.
//
// As GetSizer's documentation explains, you rarely need to care
// about or even know what sizers are.
func (self *Renderer) SetSizer(sizer esizer.Sizer) {
	if sizer == nil { panic("nil sizer") }
	self.sizer = sizer
}

// --- helper methods ---
func (self *Renderer) updateMetrics() {
	metrics := self.sizer.Metrics(self.font, self.sizePx)
	self.metrics = &metrics
}
