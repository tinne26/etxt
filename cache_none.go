package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/cache"
import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"

var _ cache.GlyphCacheHandler = (*noCacheHandler)(nil)

var pkgNoCacheHandler *noCacheHandler = &noCacheHandler{}

// A type used to allow nil caches in the renderer to be implemented
// as an empty type instead, which spares many conditionals in the code.
type noCacheHandler struct {}
func (self *noCacheHandler) NotifyFontChange(*sfnt.Font) {}
func (self *noCacheHandler) NotifySizeChange(fract.Unit) {}
func (self *noCacheHandler) NotifyRasterizerChange(mask.Rasterizer) {}
func (self *noCacheHandler) NotifyFractChange(fract.Point) {}
func (self *noCacheHandler) PassMask(sfnt.GlyphIndex, GlyphMask) {}
func (self *noCacheHandler) GetMask(sfnt.GlyphIndex) (GlyphMask, bool) {
	return nil, false
}
