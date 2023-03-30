package cache

import "sync/atomic"
import _ "unsafe"
//go:linkname systemMonoNanoTime runtime.nanotime

//go:noescape
func systemMonoNanoTime() int64

// A cached mask with additional information to estimate how
// much the entry is being used.
type cachedMaskEntry struct {
	Mask GlyphMask // Read-only.
	ByteSize uint32 // Read-only.
	CreationInstant uint32 // see cacheEntryInstant(). Read-only.
	                       // Won't be consistent after reboots. I
								  // don't know why would anyone try to
								  // save a cache, but I've seen worse.
	accessCount uint32 // number of times the entry has been accessed
}

// Must be called after accessing an entry in order to keep the
// Hotness() heuristic making sense. Concurrent-safe.
func (self *cachedMaskEntry) IncreaseAccessCount() {
	atomic.AddUint32(&self.accessCount, 1)
}

// A measure of "bytes accessed per time". Coldest entries
// (smallest values) are candidates for eviction. Concurrent-safe.
func (self *cachedMaskEntry) Hotness(instant uint32) uint32 {
	const ConstEvictionCost = 1000 // additional threshold and pad
	bytesHit := self.ByteSize*atomic.LoadUint32(&self.accessCount)
	elapsed  := instant - self.CreationInstant
	if elapsed == 0 { elapsed = 1 }
	return (ConstEvictionCost + bytesHit)/elapsed
}

// Without this in order to test some cache situations I'd need
// time.Sleep() calls, but sadly I can't make this constant
// without messing too much with tags due to what's discussed on
// https://github.com/golang/go/issues/21360. One second would
// be 1000_000_000, half a second 500_000_000, etc.
var testInstantNanosHack int64

// A time instant related to the system's monotonic nano time, but with
// some arbitrary downscaling applied (close to converting nanoseconds
// to hundredth's of seconds).
func cacheEntryInstant() uint32 {
	return uint32((systemMonoNanoTime() + testInstantNanosHack) >> 27)
}

// Creates a new cached mask entry for the given GlyphMask.
func newCachedMaskEntry(mask GlyphMask) (*cachedMaskEntry, uint32) {
	instant := cacheEntryInstant()
	return &cachedMaskEntry {
		Mask: mask,
		ByteSize: GlyphMaskByteSize(mask),
		CreationInstant: instant,
		accessCount: 1,
	}, instant
}
