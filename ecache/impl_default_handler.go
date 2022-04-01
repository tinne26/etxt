package ecache

import "unsafe"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt/emask"

// A default implementation of GlyphCacheHandler.
type DefaultCacheHandler struct {
	cache *DefaultCache
	activeKey [3]uint64
}

// Satisfies GlyphCacheHandler.NotifyFontChange(...)
func (self *DefaultCacheHandler) NotifyFontChange(font *Font) {
	self.activeKey[0] = uint64(uintptr(unsafe.Pointer(font)))
}

// Satisfies GlyphCacheHandler.NotifyRasterizerChange(...)
func (self *DefaultCacheHandler) NotifyRasterizerChange(rasterizer emask.Rasterizer) {
	self.activeKey[1] = rasterizer.CacheSignature()
}

// Satisfies GlyphCacheHandler.NotifySizeChange(...)
func (self *DefaultCacheHandler) NotifySizeChange(size fixed.Int26_6) {
	self.activeKey[2] = (self.activeKey[2] & ^uint64(0xFFFFFFFF00000000)) | (uint64(size) << 32)
}

// Satisfies GlyphCacheHandler.NotifyFractChange(...)
func (self *DefaultCacheHandler) NotifyFractChange(fract fixed.Point26_6) {
	bits := uint64(fract.Y & 0x0000003F) << 16
	bits |= uint64(fract.X & 0x0000003F) << 22
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

// Satisfies GlyphCacheHandler.GetMask(...)
func (self *DefaultCacheHandler) GetMask(index GlyphIndex) (GlyphMask, bool) {
	self.activeKey[2] = (self.activeKey[2] & ^uint64(0x000000000000FFFF)) | uint64(index)
	return self.cache.GetMask(self.activeKey)
}

// Satisfies GlyphCacheHandler.PassMask(...)
func (self *DefaultCacheHandler) PassMask(index GlyphIndex, mask GlyphMask) {
	self.activeKey[2] = (self.activeKey[2] & ^uint64(0x000000000000FFFF)) | uint64(index)
	self.cache.PassMask(self.activeKey, mask)
}

// Provides access to DefaultCache.ApproxByteSize().
func (self *DefaultCacheHandler) ApproxCacheByteSize() int {
	return self.cache.ApproxByteSize()
}

// Provides access to DefaultCache.PeakSize().
func (self *DefaultCacheHandler) PeakCacheSize() int {
	return self.cache.PeakSize()
}
