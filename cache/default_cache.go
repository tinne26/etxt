package cache

import "sync"
import "sync/atomic"

// The default etxt cache. It is concurrent-safe (though not optimized
// or expected to be used under heavily concurrent scenarios), it has
// memory bounds and uses random sampling for evicting entries.
type DefaultCache struct {
	cachedMasks map[[3]uint64]*cachedMaskEntry
	mutex       sync.RWMutex
	capacity    uint64
	currentSize uint64
	peakSize    uint64 // (max ever size)
	accessTick  uint64 // (see toNextAccessTick() for overflow details)
}

// Creates a new cache bounded by the given capacity. Negative
// values will panic.
//
// Values below 32*1024 (32KiB) are not recommended; allowing the
// cache to grow up to a few MiBs in size is generally preferable.
// For more concrete size estimations, the package overview includes
// a more detailed explanation.
func NewDefaultCache(capacityInBytes int) *DefaultCache {
	if capacityInBytes < 0 {
		panic("capacityInBytes < 0")
	} // likely a dev mistake
	return &DefaultCache{
		cachedMasks: make(map[[3]uint64]*cachedMaskEntry, 128),
		capacity:    uint64(capacityInBytes),
	}
}

func (self *DefaultCache) removeRandOldEntry() {
	// working values and variables
	const RequiredSamples = 10 // NOTE: we could make this configurable, but not a big deal
	var oldestAccess uint64 = 0xFFFF_FFFF_FFFF_FFFF
	var oldestEntryKey [3]uint64
	var entriesSampled int

	// pseudo-random entry sampling
	self.mutex.RLock()
	for key, cachedMaskEntry := range self.cachedMasks {
		access := cachedMaskEntry.LastAccess()
		if access <= oldestAccess {
			oldestAccess = access
			oldestEntryKey = key
		}

		// break if we already took enough samples
		entriesSampled += 1
		if entriesSampled >= RequiredSamples {
			break
		}
	}
	self.mutex.RUnlock()

	// delete oldest entry found
	self.mutex.Lock()
	cachedMaskEntry, stillExists := self.cachedMasks[oldestEntryKey]
	if stillExists {
		delete(self.cachedMasks, oldestEntryKey)
		maskSize := uint64(cachedMaskEntry.ByteSize())
		atomic.AddUint64(&self.currentSize, ^(maskSize - 1))
	}
	self.mutex.Unlock()
}

// Stores the given mask with the given key.
func (self *DefaultCache) PassMask(key [3]uint64, mask GlyphMask) {
	// create mask cached entry
	tick := self.toNextAccessTick()
	maskEntry := newCachedMaskEntry(mask, tick)
	maskSize := uint64(maskEntry.ByteSize())
	if maskSize > atomic.LoadUint64(&self.capacity) {
		return
	} // awkward

	// see if we have enough space to add (without concurrent checking)
	for attempt := 0; attempt < 5; attempt++ {
		if self.hasRoomForMask(maskSize) {
			break
		}
		self.removeRandOldEntry()
	}

	// add the mask to the cache
	self.mutex.Lock()
	preMask, maskAlreadyExists := self.cachedMasks[key]
	if maskAlreadyExists {
		delete(self.cachedMasks, key)
		preMaskSize := uint64(preMask.ByteSize())
		atomic.AddUint64(&self.currentSize, ^(preMaskSize - 1))
	}
	if self.hasRoomForMask(maskSize) {
		self.cachedMasks[key] = maskEntry
		newSize := atomic.AddUint64(&self.currentSize, maskSize)
		if atomic.LoadUint64(&self.peakSize) < newSize {
			atomic.StoreUint64(&self.peakSize, newSize)
		}
	}
	self.mutex.Unlock()
}

func (self *DefaultCache) hasRoomForMask(maskSize uint64) bool {
	capacity := atomic.LoadUint64(&self.capacity)
	if capacity < maskSize {
		return false
	}
	size := atomic.LoadUint64(&self.currentSize)
	return size <= capacity-maskSize
}

// Returns the capacity of the cache, in bytes.
func (self *DefaultCache) Capacity() int {
	return int(atomic.LoadUint64(&self.capacity))
}

// Returns an approximation of the number of bytes taken
// by the glyph masks currently stored in the cache.
func (self *DefaultCache) CurrentSize() int {
	return int(atomic.LoadUint64(&self.currentSize))
}

func (self *DefaultCache) remainingCapacity() uint64 {
	currSize := atomic.LoadUint64(&self.currentSize)
	capacity := atomic.LoadUint64(&self.capacity)
	if currSize >= capacity {
		return 0
	}
	return capacity - currSize
}

// Returns an approximation of the maximum amount of bytes that
// the cache has been filled with throughout its life.
//
// This method can be useful to determine the actual usage of a cache
// within your application and set its capacity to a reasonable value.
func (self *DefaultCache) PeakSize() int {
	return int(atomic.LoadUint64(&self.peakSize))
}

// Returns the number of cached masks currently in the cache.
func (self *DefaultCache) NumEntries() int {
	self.mutex.RLock()
	numEntries := len(self.cachedMasks)
	self.mutex.RUnlock()
	return numEntries
}

// Gets the mask associated to the given key.
func (self *DefaultCache) GetMask(key [3]uint64) (GlyphMask, bool) {
	self.mutex.RLock()
	entry, found := self.cachedMasks[key]
	self.mutex.RUnlock()
	if !found {
		return nil, false
	}

	tick := self.toNextAccessTick()
	entry.UpdateAccess(tick)
	return entry.Mask, true
}

// Like GetMask, but doesn't update the last access for the mask
// on the cache. Used for debugging, where we sometimes need to observe
// without causing side-effects.
func (self *DefaultCache) sneakyGetMask(key [3]uint64) (GlyphMask, bool) {
	self.mutex.RLock()
	entry, found := self.cachedMasks[key]
	self.mutex.RUnlock()
	return entry.Mask, found
}

func (self *DefaultCache) toNextAccessTick() uint64 {
	tick := atomic.AddUint64(&self.accessTick, 1)
	if tick == 0 {
		// human error is much more likely than the event itself,
		// so panicking makes more sense than a fallback (which
		// would also be easy to write in three lines)
		panic("broken code (or +243k years elapsed drawing 10k glyphs per frame at 240fps)")
	}
	return tick
}

// Returns a new cache handler for the current cache. While DefaultCache
// is concurrent-safe, handlers can only be used non-concurrently. One
// can create multiple handlers for the same cache to be used with different
// renderers.
func (self *DefaultCache) NewHandler() *DefaultCacheHandler {
	var zeroKey [3]uint64
	return &DefaultCacheHandler{cache: self, activeKey: zeroKey}
}
