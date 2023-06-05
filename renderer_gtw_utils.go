package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/cache"

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

// Utility method to set the font by passing its raw data and letting
// the renderer parse it. This method should be avoided if you want
// to reuse the font data at different points in your application; in
// that case, parsing the font only once and setting it with
// [Renderer.SetFont]() is the way to go.
//
// For further font management functionality, consider [font.Library].
//
// [font.Library]: https://pkg.go.dev/github.com/tinne26/etxt/font
func (self *RendererUtils) SetFontBytes(data []byte) error {
	return (*Renderer)(self).utilsSetFontBytes(data)
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
	return self.GetSizer().LineHeight(self.GetFont(), &self.buffer, self.fractGetScaledSize()).ToFloat64()
}
