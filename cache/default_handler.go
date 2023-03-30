package cache

import "unsafe"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"

var _ GlyphCacheHandler = (*DefaultCacheHandler)(nil)

// A default implementation of [GlyphCacheHandler].
type DefaultCacheHandler struct {
	cache *DefaultCache
	activeKey [3]uint64
}

// Implements [GlyphCacheHandler].NotifyFontChange(...)
func (self *DefaultCacheHandler) NotifyFontChange(font *sfnt.Font) {
	self.activeKey[0] = uint64(uintptr(unsafe.Pointer(font)))
}

// Implements [GlyphCacheHandler].NotifyRasterizerChange(...)
func (self *DefaultCacheHandler) NotifyRasterizerChange(rasterizer mask.Rasterizer) {
	self.activeKey[1] = rasterizer.Signature()
}

// Implements [GlyphCacheHandler].NotifySizeChange(...)
func (self *DefaultCacheHandler) NotifySizeChange(size fract.Unit) {
	self.activeKey[2] = (self.activeKey[2] & ^uint64(0xFFFFFFFF00000000)) | (uint64(size) << 32)
}

// Implements [GlyphCacheHandler].NotifyFractChange(...)
func (self *DefaultCacheHandler) NotifyFractChange(fract fract.Point) {
	bits := uint64(fract.Y.FractShift()) << 16
	bits |= uint64(fract.X.FractShift()) << 22
	self.activeKey[2] = (self.activeKey[2] & ^uint64(0x000000000FFF0000)) | bits
}

// This is not a thing nowadays, but if sfnt ever implemented proper hinting
// and you could detect whether a glyph mask has hinting instructions applied
// or not, or if you implemented some other hinting mechanism yourself, you
// could use this "variant" change to differentiate the glyphs. This code
// only allows 4 bits to encode variants, but since etxt.Renderer doesn't
// use all the bits from the size, we could easily shave ~12 bits more from
// the size key encoding and go up to 16 bits for variants.
//
// For rasterizer-based hinting it doesn't matter much, though, as the 64
// bits from their cache signature can also do the job.
// func (self *DefaultCacheHandler) NotifyVariantChange(variant uint8) {
// 	self.activeKey[2] = (self.activeKey[2] & ^uint64(0x00000000F0000000)) | (uint64(variant ^ 0x0F) << 28)
// }

// Implements [GlyphCacheHandler].GetMask(...)
func (self *DefaultCacheHandler) GetMask(index sfnt.GlyphIndex) (GlyphMask, bool) {
	self.activeKey[2] = (self.activeKey[2] & ^uint64(0x000000000000FFFF)) | uint64(index)
	return self.cache.GetMask(self.activeKey)
}

// Implements [GlyphCacheHandler].PassMask(...)
func (self *DefaultCacheHandler) PassMask(index sfnt.GlyphIndex, mask GlyphMask) {
	self.activeKey[2] = (self.activeKey[2] & ^uint64(0x000000000000FFFF)) | uint64(index)
	self.cache.PassMask(self.activeKey, mask)
}

// Provides access to [DefaultCache.ApproxByteSize]().
func (self *DefaultCacheHandler) ApproxCacheByteSize() int {
	return self.cache.ApproxByteSize()
}

// Provides access to [DefaultCache.PeakSize]().
func (self *DefaultCacheHandler) PeakCacheSize() int {
	return self.cache.PeakSize()
}

// Provides access to the underlying [DefaultCache].
func (self *DefaultCacheHandler) Cache() *DefaultCache {
	return self.cache
}
