package etxt

import (
	"image/color"
	"strconv"

	"github.com/tinne26/etxt/cache"
	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/etxt/mask"
	"github.com/tinne26/etxt/sizer"
	"golang.org/x/image/font/sfnt"
)

// [Gateway] to [RendererUtils] functionality.
//
// [Gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9#Renderer
func (self *Renderer) Utils() *RendererUtils {
	return (*RendererUtils)(self)
}

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to utility functions for a [Renderer].
//
// In general, this type is used through method chaining:
//
//	renderer.Utils().SetCache8MiB()
//
// The focus of this type are non-essential but handy functions
// that can make it easier to set up [Renderer] properties or
// perform certain operations. Most programs on [etxt/examples/ebiten]
// make use of this.
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9#Renderer
// [etxt/examples/ebiten]: https://github.com/tinne26/etxt/tree/v0.0.9/examples/ebiten
type RendererUtils Renderer

// ---- wrapper methods ----

// Related to [RendererUtils.SetCache8MiB]().
var pkgCache8MiB *cache.DefaultCache

// Utility method to set a cache that will get you started.
// For a more manual and adjustable approach, see
// [Renderer.SetCacheHandler]() instead.
func (self *RendererUtils) SetCache8MiB() {
	(*Renderer)(self).utilsSetCache8MiB()
}

// Utility method to get the current line height. Functionally equivalent to:
//
//	font   := renderer.GetFont()
//	buffer := renderer.GetBuffer()
//	size   := renderer.Fract().GetScaledSize()
//	sizer  := renderer.GetSizer()
//	lineHeight := sizer.LineHeight(font, buffer, size).ToFloat64()
func (self *RendererUtils) GetLineHeight() float64 {
	return (*Renderer)(self).utilsGetLineHeight()
}

// Utility method that returns the signed distance between the baseline
// and the given vertical align anchor for the current text configuration.
//
// You can expect the offset from Top to Baseline to be greater than the
// offset from MidLine to Baseline, the offset from Baseline to Baseline
// to be always 0, and the offset from Bottom to Baseline to be a negative
// value smaller in magnitude than Top to Baseline (normally).
//
// Notice that VertCenter and Bottom will assume the height of a single line,
// and LastBaseline will return 0 like Baseline.
func (self *RendererUtils) GetDistToBaseline(vertAlign Align) float64 {
	return (*Renderer)(self).utilsGetDistToBaseline(vertAlign)
}

// Sets default values for any uninitialized properties that are
// required to make the renderer produce visible results, except
// for the font. Notice that this also excludes the cache handler.
//
// This function exists solely to soothe troubled souls; solid
// reasons to use it in practice might never be found.
func (self *RendererUtils) FillMissingProperties() {
	(*Renderer)(self).utilsFillMissingProperties()
}

// Utility method to set the font by passing its raw data and letting
// the renderer parse it. This method should be avoided if you want
// to reuse the font data at different points in your application; in
// that case, parsing the font only once and setting it with
// [Renderer.SetFont]() is the way to go.
//
// For more advanced font management functionality, see
// [etxt/font.Library].
//
// [etxt/font.Library]: https://pkg.go.dev/github.com/tinne26/etxt/font@v0.0.9#Library
func (self *RendererUtils) SetFontBytes(data []byte) error {
	return (*Renderer)(self).utilsSetFontBytes(data)
}

// Makes a copy of the current renderer state and pushes it
// into an internal stack. Stored states can be recovered with
// [RendererUtils.RestoreState]() in last-in first-out order.
//
// The stored state includes the following properties:
//   - [Align], color, size, scale, [BlendMode], rasterizer,
//     sizer, quantization and text [Direction].
//
// Notably, custom rendering functions, inactive fonts
// and the cache handler are not stored.
//
// For improved safety when managing states, consider also
// [RendererUtils.AssertMaxStoredStates]().
func (self *RendererUtils) StoreState() {
	(*Renderer)(self).utilsStoreState()
}

// The counterpart of [RendererUtils.StoreState](). Restores the
// most recently stored renderer state and removes it from the
// internal stack.
//
// Iff the states stack is empty and no state restoration is
// possible, the function returns false.
func (self *RendererUtils) RestoreState() bool {
	return (*Renderer)(self).utilsRestoreState()
}

// Panics when the size of the stored states stack exceeds
// the given value. States are stored with [RendererUtils.StoreState]().
//
// There are two main ways to use this function:
//   - Regularly asserting that the number of stored states
//     stays below a reasonable limit for your use-case in
//     order to prevent memory leaks.
//   - Passing zero whenever you want to ensure that the
//     states stack is completely empty.
func (self *RendererUtils) AssertMaxStoredStates(n int) {
	if n > len(self.restorableStates) {
		assertMax := strconv.Itoa(n)
		actualMax := strconv.Itoa(len(self.restorableStates))
		panic("expected at most " + assertMax + " stored states, found " + actualMax)
	}
}

// ---- underlying implementations ----

func (self *Renderer) utilsSetCache8MiB() {
	// This uses a package level cache that will be shared by
	// all renderers using this utility method. If you have
	// many renderers with different fonts, it may be better
	// to start creating your own caches.
	if pkgCache8MiB == nil {
		pkgCache8MiB = cache.NewDefaultCache(8 * 1024 * 1024)
	}
	self.SetCacheHandler(pkgCache8MiB.NewHandler())
}

func (self *Renderer) utilsSetFontBytes(data []byte) error {
	font, err := sfnt.Parse(data)
	if err != nil {
		return err
	}
	self.SetFont(font)
	return nil
}

func (self *Renderer) utilsGetLineHeight() float64 {
	return self.state.fontSizer.LineHeight(self.state.activeFont, &self.buffer, self.state.scaledSize).ToFloat64()
}

func (self *Renderer) utilsGetDistToBaseline(vertAlign Align) float64 {
	return self.getDistToBaselineFract(vertAlign).ToFloat64()
}

func (self *Renderer) getDistToBaselineFract(vertAlign Align) fract.Unit {
	switch vertAlign.Vert() {
	case Top:
		return self.getOpAscent()
	case CapLine:
		return self.getSlowOpCapHeight()
	case Midline:
		return self.getSlowOpXHeight()
	case VertCenter:
		return self.getOpAscent() - (self.getOpLineHeight() >> 1)
	case Baseline:
		return 0
	case LastBaseline:
		return 0
	case Bottom:
		return self.getOpAscent() - self.getOpLineHeight()
	default:
		panic(vertAlign)
	}
}

func (self *Renderer) utilsFillMissingProperties() {
	if self.state.rasterizer == nil {
		self.glyphSetRasterizer(&mask.DefaultRasterizer{})
	}
	if self.state.fontColor == nil {
		self.state.fontColor = color.RGBA{255, 255, 255, 255}
	}
	if self.state.horzQuantization == 0 {
		self.state.horzQuantization = uint8(Qt4th)
	}
	if self.state.vertQuantization == 0 {
		self.state.vertQuantization = uint8(QtFull)
	}
	if self.state.align&alignVertBits == 0 {
		self.state.align = self.state.align | Baseline
	}
	if self.state.align&alignHorzBits == 0 {
		self.state.align = self.state.align | Left
	}

	var refreshSize bool
	if self.state.scale == 0 {
		self.state.scale = 64
		refreshSize = true
	}
	if self.state.logicalSize == 0 {
		self.state.logicalSize = 16 * fract.One
		refreshSize = true
	}
	if refreshSize {
		self.refreshScaledSize() // also notifies the cache handler and sizer
	}

	if self.state.fontSizer == nil {
		self.state.fontSizer = &sizer.DefaultSizer{}
		self.state.fontSizer.NotifyChange(self.state.activeFont, &self.buffer, self.state.scaledSize)
	}
}

func (self *Renderer) utilsStoreState() {
	self.restorableStates = append(self.restorableStates, self.state)
}

func (self *Renderer) utilsRestoreState() bool {
	if len(self.restorableStates) == 0 {
		return false
	}
	last := len(self.restorableStates) - 1
	self.setState(self.restorableStates[last])
	self.restorableStates = self.restorableStates[0:last]
	return true
}

func (self *Renderer) setState(state restorableState) {
	initFont := self.state.activeFont
	initSizer := self.state.fontSizer
	initSize := self.state.scaledSize
	initRast := self.state.rasterizer

	self.state = state

	// notify changes where relevant
	refreshSizer := (self.state.fontSizer != initSizer)
	if self.state.scaledSize != initSize {
		refreshSizer = true
		if self.cacheHandler != nil {
			self.cacheHandler.NotifySizeChange(self.state.scaledSize)
		}
	}
	if initFont != self.state.activeFont {
		refreshSizer = true
		if self.cacheHandler != nil {
			self.cacheHandler.NotifyFontChange(self.state.activeFont)
		}
	}

	if refreshSizer && self.state.fontSizer != nil {
		self.state.fontSizer.NotifyChange(self.state.activeFont, &self.buffer, self.state.scaledSize)
	}

	if self.state.rasterizer != initRast {
		// clear previous rasterizer onChangeFunc
		if initRast != nil {
			initRast.SetOnChangeFunc(nil)
		}

		// link new rasterizer to the cache handler
		if self.cacheHandler == nil {
			self.state.rasterizer.SetOnChangeFunc(nil)
		} else {
			self.state.rasterizer.SetOnChangeFunc(self.cacheHandler.NotifyRasterizerChange)
			self.cacheHandler.NotifyRasterizerChange(self.state.rasterizer)
		}
	}
}
