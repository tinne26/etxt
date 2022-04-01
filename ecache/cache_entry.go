package ecache

import "sync/atomic"
import _ "unsafe"
//go:linkname systemMonoNanoTime runtime.nanotime

//go:noescape
func systemMonoNanoTime() int64

// A cached mask with additional information to estimate how
// much the entry is being used. Cache implementers may use this
// type to make their life easier.
type CachedMaskEntry struct {
	Mask GlyphMask // Read-only.
	ByteSize uint32 // Read-only.
	CreationInstant uint32 // see CacheEntryInstant. Read-only.
	                       // Won't be consistent after reboots. I
								  // don't know why would anyone try to
								  // save a cache, but I've seen worse.
	accessCount uint32 // number of times the entry has been accessed
}

// Must be called after accessing an entry in order to keep the
// Hotness() heuristic making sense. Concurrent-safe.
func (self *CachedMaskEntry) IncreaseAccessCount() {
	atomic.AddUint32(&self.accessCount, 1)
}

// A measure of "bytes accessed per time". Coldest entries
// (smallest values) are candidates for eviction. Concurrent-safe.
func (self *CachedMaskEntry) Hotness(instant uint32) uint32 {
	const ConstEvictionCost = 1000 // additional threshold and pad
	bytesHit := self.ByteSize*atomic.LoadUint32(&self.accessCount)
	elapsed  := instant - self.CreationInstant
	if elapsed == 0 { elapsed = 1 }
	return (ConstEvictionCost + bytesHit)/elapsed
}

// A time instant related to the system's monotonic nano time, but with
// some arbitrary downscaling applied (close to converting nanoseconds
// to hundredth's of seconds).
func CacheEntryInstant() uint32 {
	return uint32(systemMonoNanoTime() >> 27)
}

// Creates a new cached mask entry for the given GlyphMask.
func NewCachedMaskEntry(mask GlyphMask) (*CachedMaskEntry, uint32) {
	instant := CacheEntryInstant()
	return &CachedMaskEntry {
		Mask: mask,
		ByteSize: GlyphMaskByteSize(mask),
		CreationInstant: instant,
		accessCount: 1,
	}, instant
}
