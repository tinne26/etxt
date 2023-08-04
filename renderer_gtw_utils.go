package etxt

import "strconv"
import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/cache"
import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/sizer"

// [Gateway] to [RendererUtils] functionality.
//
// [Gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
func (self *Renderer) Utils() *RendererUtils {
	return (*RendererUtils)(self)
}

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to utility [Renderer] functions.
//
// In general, this type is used through method chaining:
//   renderer.Utils().SetCache8MiB()
//
// The focus of this type are non-essential but handy functions
// that can make it easier to set up the [Renderer] properties.
// To put an example, most programs on [etxt/examples/ebiten]
// make use of these.
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
// [examples/ebiten]: https://github.com/tinne26/etxt/tree/main/examples/ebiten
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

// Utility method to get the current line height. Equivalent to:
//   buffer := renderer.GetBuffer()
//   font   := renderer.GetFont()
//   size   := renderer.Fract().GetScaledSize()
//   sizer  := renderer.GetSizer()
//   lineHeight := sizer.LineHeight(buffer, font, size).ToFloat64()
func (self *RendererUtils) GetLineHeight() float64 {
	return (*Renderer)(self).utilsGetLineHeight()
}

// This function only exists to soothe troubled souls.
// In practice, there's no rational reason to ever use it.
//
// Sets default values for any uninitialized properties that are
// required to make the renderer produce visible results, except
// for the font. Notice that the cache handler isn't included.
func (self *RendererUtils) FillMissingProperties() {
	(*Renderer)(self).utilsFillMissingProperties()
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
	if self.state.align & alignVertBits == 0 {
		self.state.align = self.state.align | Baseline
	}
	if self.state.align & alignHorzBits == 0 {
		self.state.align = self.state.align | Left
	}

	var refreshSize bool
	if self.state.scale == 0 {
		self.state.scale = 64
		refreshSize = true
	}
	if self.state.logicalSize == 0 {
		self.state.logicalSize = 16*fract.One
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

// Utility method to set the font by passing its raw data and letting
// the renderer parse it. This method should be avoided if you want
// to reuse the font data at different points in your application; in
// that case, parsing the font only once and setting it with
// [Renderer.SetFont]() is the way to go.
//
// For more advanced font management functionality, see
// [github.com/tinne26/etxt/font.Library].
func (self *RendererUtils) SetFontBytes(data []byte) error {
	return (*Renderer)(self).utilsSetFontBytes(data)
}

// Makes a copy of the current renderer state and pushes it
// into an internal stack. Stored states can be recovered with
// [RendererUtils.RestoreState]() in last-in first-out order.
// 
// The stored state includes the following properties:
//  - [Align], color, size, scale, [BlendMode], [FontIndex],
//    active font, rasterizer, sizer, quantization and
//    text [Direction].
// Notably, the custom rendering function, the inactive font set
// and the cache handler are not stored.
//
// For improved safety when storing states, consider looking
// into [RendererUtils.AssertMaxStoredStates]().
func (self *RendererUtils) StoreState() {
	(*Renderer)(self).utilsStoreState()
}

// The counterpart of [RendererUtils.StoreState](). Restores the
// most recently stored renderer state and removes it from the
// internal stack.
//
// If the states stack is empty and no state restoration is
// possible, this function returns false.
func (self *RendererUtils) RestoreState() bool {
	return (*Renderer)(self).utilsRestoreState()
}

// Panics when the size of the stored states stack exceeds
// the given value. States are stored with [RendererUtils.StoreState]().
// 
// There are two main ways to use this function:
//  - Regularly asserting that the number of stored states
//    stays below a reasonable limit for your use-case in
//    order to prevent memory leaks.
//  - Passing zero to the function whenever you want to ensure
//    that the states stack is completely empty.
func (self *RendererUtils) AssertMaxStoredStates(n int) {
	if n > len(self.restorableStates) {
		givenMax  := strconv.Itoa(n)
		actualMax := strconv.Itoa(len(self.restorableStates))
		panic("expected at most " + givenMax + " stored states, found " + actualMax)
	}
}

// ---- underlying implementations ----

func (self *Renderer) utilsSetCache8MiB() {
	// This uses a package level cache that will be shared by
	// all renderers using this utility method. If you have
	// many renderers with different fonts, it may be better
	// to start creating your own caches.
	if pkgCache8MiB == nil {
		pkgCache8MiB = cache.NewDefaultCache(8*1024*1024)
	}
	self.SetCacheHandler(pkgCache8MiB.NewHandler())
}

func (self *Renderer) utilsSetFontBytes(data []byte) error {
	font, err := sfnt.Parse(data)
	if err != nil { return err }
	self.SetFont(font)
	return nil
}

func (self *Renderer) utilsGetLineHeight() float64 {
	return self.state.fontSizer.LineHeight(self.state.activeFont, &self.buffer, self.state.scaledSize).ToFloat64()
}

func (self *Renderer) utilsStoreState() {
	self.restorableStates = append(self.restorableStates, self.state)
}

func (self *Renderer) utilsRestoreState() bool {
	if len(self.restorableStates) == 0 { return false }
	
	initFont  := self.state.activeFont
	initSizer := self.state.fontSizer
	initSize  := self.state.scaledSize
	initRast  := self.state.rasterizer
	
	last := len(self.restorableStates) - 1
	self.state = self.restorableStates[last]
	self.restorableStates = self.restorableStates[0 : last]

	// notify changes where relevant
	refreshSizer := (self.state.scaledSize != initSize || self.state.fontSizer != initSizer)
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
		if initRast != nil { initRast.SetOnChangeFunc(nil) }

		// link new rasterizer to the cache handler
		if self.cacheHandler == nil {
			self.state.rasterizer.SetOnChangeFunc(nil)
		} else {
			self.state.rasterizer.SetOnChangeFunc(self.cacheHandler.NotifyRasterizerChange)
			self.cacheHandler.NotifyRasterizerChange(self.state.rasterizer)
		}
	}
	
	return true
}
