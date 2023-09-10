package cache

import "sync/atomic"

// A cached mask with additional information to estimate how
// much the entry is being used.
type cachedMaskEntry struct {
	Mask GlyphMask // Read-only.
	lastAccess uint64
	byteSize uint32 // Read-only.
}

func (self *cachedMaskEntry) UpdateAccess(accessTick uint64) {
	atomic.StoreUint64(&self.lastAccess, accessTick)
}

func (self *cachedMaskEntry) LastAccess() uint64 {
	return atomic.LoadUint64(&self.lastAccess)
}

func (self *cachedMaskEntry) ByteSize() uint32 {
	return atomic.LoadUint32(&self.byteSize)
}

// Creates a new cached mask entry for the given GlyphMask.
func newCachedMaskEntry(mask GlyphMask, accessTick uint64) *cachedMaskEntry {
	return &cachedMaskEntry {
		Mask: mask,
		lastAccess: accessTick,
		byteSize: GlyphMaskByteSize(mask),
	}
}
