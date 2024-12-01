package cache

import "sort"
import "strconv"
import "testing"

// Debug method, e.g.:
//
//	t.Fatalf("debug cache keys:\n%s", cache.getKeysAsDebugStr())
func (self *DefaultCache) getKeysAsDebugStr() string {
	strList := make([]string, 0, 8)
	charCount := 0
	self.mutex.RLock()
	for key, _ := range self.cachedMasks {
		str := strconv.FormatUint(key[0], 10) + "-" + strconv.FormatUint(key[1], 10) + "-" + strconv.FormatUint(key[2], 10)
		charCount += len(str) + 1
		strList = append(strList, str)
	}
	self.mutex.RUnlock()

	charCount -= 1
	sort.Strings(strList)
	result := make([]byte, 0, charCount)
	for i, str := range strList {
		result = append(result, str...)
		if i != len(strList)-1 {
			result = append(result, '\n')
		}
	}
	return string(result)
}

func TestDefaultCache(t *testing.T) {
	masks := make([]GlyphMask, 10)
	for i := 0; i < 10; i++ {
		masks[i] = newEmptyGlyphMask(10, 10)
	}
	refSize := GlyphMaskByteSize(masks[0])

	cache := NewDefaultCache(int(refSize * 8))
	gotSize := cache.CurrentSize()
	if gotSize != 0 {
		t.Fatalf("expected %d, got %d", 0, gotSize)
	}

	gotSize = cache.PeakSize()
	if gotSize != 0 {
		t.Fatalf("expected %d, got %d", 0, gotSize)
	}

	mask, found := cache.GetMask([3]uint64{0, 0, 1})
	if found {
		t.Fatal("didn't expect to find mask")
	}
	if mask != nil {
		t.Fatal("expected nil mask")
	}

	cache.PassMask([3]uint64{0, 0, 2}, masks[2])
	_, found = cache.GetMask([3]uint64{0, 0, 1})
	if found {
		t.Fatal("didn't expect to find mask")
	}

	mask, found = cache.GetMask([3]uint64{0, 0, 2})
	if !found {
		t.Fatal("expected to find mask")
	}
	if masks[2] != mask {
		t.Fatal("nonsensical mask")
	}

	for i := 3; i < 10; i++ {
		cache.PassMask([3]uint64{0, 0, uint64(i)}, masks[i])
	}

	for i := 3; i < 10; i++ {
		mask, found = cache.GetMask([3]uint64{0, 0, uint64(i)})
		if !found {
			t.Fatal("expected to find mask")
		}
		if masks[i] != mask {
			t.Fatal("wrong mask")
		}
		if masks[i-1] == mask {
			t.Fatal("broken test")
		}
	}

	gotSize = cache.CurrentSize()
	expectSize := int(refSize) * 8
	if gotSize != expectSize {
		t.Fatalf("expected %d, got %d", expectSize, gotSize)
	}
	gotSize = cache.PeakSize()
	if gotSize != expectSize {
		t.Fatalf("expected %d, got %d", expectSize, gotSize)
	}

	cache.PassMask([3]uint64{0, 0, 0}, masks[0])
	mask, found = cache.GetMask([3]uint64{0, 0, 0})
	if !found {
		t.Fatal("expected mask to be added")
	}

	mask, found = cache.GetMask([3]uint64{0, 0, 2})
	if found {
		t.Fatal("expected mask to be evicted")
	}

	gotSize = cache.CurrentSize()
	if gotSize != expectSize {
		t.Fatalf("expected %d, got %d", expectSize, gotSize)
	}
	gotSize = cache.PeakSize()
	if gotSize != expectSize {
		t.Fatalf("expected %d, got %d", expectSize, gotSize)
	}
	for i := 3; i < 10; i++ {
		mask, found = cache.sneakyGetMask([3]uint64{0, 0, uint64(i)})
		if !found {
			t.Fatal("expected to find mask")
		}
		if masks[i] != mask {
			t.Fatal("wrong mask")
		}
		if masks[i-1] == mask {
			t.Fatal("broken test")
		}
	}

	biggerMask := newEmptyGlyphMask(12, 12)
	cache.PassMask([3]uint64{999, 999, 999}, biggerMask)
	_, found = cache.GetMask([3]uint64{0, 0, 3})
	if found {
		t.Fatal("expected mask to be evicted")
	}
	_, found = cache.GetMask([3]uint64{0, 0, 4})
	if found {
		t.Fatal("expected mask to be evicted")
	} // TODO: why?
	_, found = cache.GetMask([3]uint64{0, 0, 1})
	if found {
		t.Fatal("this mask hasn't been ever passed to cache")
	}
	_, found = cache.GetMask([3]uint64{0, 0, 5})
	if !found {
		t.Fatal("expected mask to be present")
	}
	_, found = cache.GetMask([3]uint64{0, 0, 0})
	if !found {
		t.Fatal("expected mask to be present")
	}

	gotSize = cache.PeakSize()
	if gotSize != expectSize {
		t.Fatalf("expected %d, got %d", expectSize, gotSize)
	}
	expectSize = expectSize - int(refSize*2) + int(GlyphMaskByteSize(biggerMask))
	gotSize = cache.CurrentSize()
	if gotSize != expectSize {
		t.Fatalf("expected %d, got %d", expectSize, gotSize)
	}
}
