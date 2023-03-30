package cache

import "sync"
import "sync/atomic"
import "math/rand"

// The default etxt cache. It is concurrent-safe (though not optimized
// or expected to be used under heavily concurrent scenarios), it has
// memory bounds and uses random sampling for evicting entries.
type DefaultCache struct {
	cachedMasks map[[3]uint64]*cachedMaskEntry
	rng *rand.Rand
	spaceBytesLeft uint32
	lowestBytesLeft uint32
	byteSizeLimit uint32
	mutex sync.RWMutex
}

// Creates a new cache bounded by the given size. Negative values
// will panic.
//
// Values below 32*1024 (32KiB) are not recommended; allowing the
// cache to grow up to a few MiBs in size is generally preferable.
// For more concrete size estimations, the package overview includes
// a more detailed explanation.
func NewDefaultCache(maxByteSize int) *DefaultCache {
	if maxByteSize < 0 { panic("maxByteSize < 0") } // likely a dev mistake
	return &DefaultCache {
		cachedMasks: make(map[[3]uint64]*cachedMaskEntry, 128),
		spaceBytesLeft: uint32(maxByteSize),
		lowestBytesLeft: uint32(maxByteSize),
		byteSizeLimit: uint32(maxByteSize),
		rng: rand.New(rand.NewSource(systemMonoNanoTime()^0x36285016_051A1E33)),
	}
}

// Attempts to remove the entry with the lowest eviction cost from a
// small pool of samples. May not remove anything in some cases.
//
// The returned value is the freed space, which must be manually
// added to spaceBytesLeft by the caller.
func (self *DefaultCache) removeRandEntry(hotness uint32, instant uint32) uint32 {
	const SampleSize = 10 // NOTE: we could make this configurable, but not a big deal

	self.mutex.RLock()
	var selectedKey [3]uint64
	lowestHotness := ^uint32(0)
	samplesTaken  := 0
	for key, cachedMaskEntry := range self.cachedMasks {
		currHotness := cachedMaskEntry.Hotness(instant)
		// on lower hotness, update selected eviction target
		if currHotness < lowestHotness {
			lowestHotness = currHotness
			selectedKey = key
		}

		// break if we already took enough samples
		samplesTaken += 1
		if samplesTaken >= SampleSize { break }
	}
	self.mutex.RUnlock()

	// delete selected entry, if any
	freedSpace := uint32(0)
	if lowestHotness < hotness {
		self.mutex.Lock()
		entry, stillExists := self.cachedMasks[selectedKey]
		if stillExists {
			delete(self.cachedMasks, selectedKey)
			freedSpace = entry.ByteSize
		}
		self.mutex.Unlock()
	}
	return freedSpace
}

// Stores the given mask with the given key.
func (self *DefaultCache) PassMask(key [3]uint64, mask GlyphMask) {
	const MaxMakeRoomAttempts = 2

	// see if we have enough space to add the mask, or try to
	// make some room otherwise
	maskEntry, instant := newCachedMaskEntry(mask)
	if maskEntry.ByteSize > atomic.LoadUint32(&self.byteSizeLimit) { return }
	spaceBytesLeft := atomic.LoadUint32(&self.spaceBytesLeft)
	freedSpace := uint32(0)
	if maskEntry.ByteSize > spaceBytesLeft {
		hotness := maskEntry.Hotness(instant)
		missingSpace := maskEntry.ByteSize - spaceBytesLeft
		for i := 0; i < MaxMakeRoomAttempts; i++ {
			freedSpace += self.removeRandEntry(hotness, instant)
			if freedSpace >= missingSpace { goto roomMade }
		}

		// we didn't make enough room for the new entry. desist.
		if freedSpace != 0 {
			atomic.AddUint32(&self.spaceBytesLeft, freedSpace)
		}
		return
	}

roomMade:
	// add the mask to the cache
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if freedSpace != 0 { atomic.AddUint32(&self.spaceBytesLeft, freedSpace) }
	_, maskAlreadyExists := self.cachedMasks[key]
	if maskAlreadyExists { return }
	if atomic.LoadUint32(&self.spaceBytesLeft) < maskEntry.ByteSize { return }
	newLeft := atomic.AddUint32(&self.spaceBytesLeft, ^uint32(maskEntry.ByteSize - 1))
	if newLeft < atomic.LoadUint32(&self.lowestBytesLeft) {
		atomic.StoreUint32(&self.lowestBytesLeft, newLeft)
	}
	self.cachedMasks[key] = maskEntry
}

// Returns an approximation of the number of bytes taken by the
// glyph masks currently stored in the cache.
func (self *DefaultCache) ApproxByteSize() int {
	return int(atomic.LoadUint32(&self.byteSizeLimit) - atomic.LoadUint32(&self.spaceBytesLeft))
}

// Returns an approximation of the maximum amount of bytes that the
// cache has been filled with at any point of its life.
//
// This method can be useful to determine the actual usage of a cache
// within your application and set its capacity to a reasonable value.
func (self *DefaultCache) PeakSize() int {
	return int(atomic.LoadUint32(&self.byteSizeLimit) - atomic.LoadUint32(&self.lowestBytesLeft))
}

// Gets the mask associated to the given key.
func (self *DefaultCache) GetMask(key [3]uint64) (GlyphMask, bool) {
	self.mutex.RLock()
	entry, found := self.cachedMasks[key]
	self.mutex.RUnlock()
	if !found { return nil, false }
	entry.IncreaseAccessCount()
	return entry.Mask, true
}

// Returns a new cache handler for the current cache. While DefaultCache
// is concurrent-safe, handlers can only be used non-concurrently. One
// can create multiple handlers for the same cache to be used with different
// renderers.
func (self *DefaultCache) NewHandler() *DefaultCacheHandler {
	var zeroKey [3]uint64
	return &DefaultCacheHandler { cache: self, activeKey: zeroKey }
}
