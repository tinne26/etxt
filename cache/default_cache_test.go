package cache

import "testing"

func TestDefaultCache(t *testing.T) {
	const millis200 = 200_000_000 // 200 milliseconds in ns

	masks := make([]GlyphMask, 10)
	for i := 0; i < 10; i++ {
		masks[i] = newEmptyGlyphMask(10, 10)
	}
	refSize := GlyphMaskByteSize(masks[0])

	cache := NewDefaultCache(int(refSize*8))
	gotSize := cache.ApproxByteSize()
	if gotSize != 0 { t.Fatalf("expected %d, got %d", 0, gotSize) }

	gotSize  = cache.PeakSize()
	if gotSize != 0 { t.Fatalf("expected %d, got %d", 0, gotSize) }

	mask, found := cache.GetMask([3]uint64{0, 0, 1})
	if found { t.Fatal("didn't expect to find mask") }
	if mask != nil { t.Fatal("expected nil mask") }

	cache.PassMask([3]uint64{0, 0, 2}, masks[2])
	_, found = cache.GetMask([3]uint64{0, 0, 1})
	if found { t.Fatal("didn't expect to find mask") }

	mask, found = cache.GetMask([3]uint64{0, 0, 2})
	if !found { t.Fatal("expected to find mask") }
	if masks[2] != mask { t.Fatal("nonsensical mask") }

	for i := 3; i < 10; i++ {
		if i <= 5 { testInstantNanosHack += millis200 } // keep mask additions appart
		cache.PassMask([3]uint64{0, 0, uint64(i)}, masks[i])
	}

	for i := 3; i < 10; i++ {
		mask, found = cache.GetMask([3]uint64{0, 0, uint64(i)})
		if !found { t.Fatal("expected to find mask") }
		if masks[i] != mask { t.Fatal("wrong mask") }
		if masks[i - 1] == mask { t.Fatal("broken test") }
	}

	gotSize = cache.ApproxByteSize()
	expectSize := int(refSize)*8
	if gotSize != expectSize { t.Fatalf("expected %d, got %d", expectSize, gotSize) }
	gotSize = cache.PeakSize()
	if gotSize != expectSize { t.Fatalf("expected %d, got %d", expectSize, gotSize) }

	testInstantNanosHack += millis200
	cache.PassMask([3]uint64{0, 0, 0}, masks[0])
	mask, found = cache.GetMask([3]uint64{0, 0, 0})
	if !found { t.Fatal("expected mask to be added") }

	mask, found = cache.GetMask([3]uint64{0, 0, 2})
	if found { t.Fatal("expected mask to be evicted") }

	gotSize = cache.ApproxByteSize()
	if gotSize != expectSize { t.Fatalf("expected %d, got %d", expectSize, gotSize) }
	gotSize = cache.PeakSize()
	if gotSize != expectSize { t.Fatalf("expected %d, got %d", expectSize, gotSize) }

	for i := 3; i < 10; i++ {
		mask, found = cache.GetMask([3]uint64{0, 0, uint64(i)})
		if !found { t.Fatal("expected to find mask") }
		if masks[i] != mask { t.Fatal("wrong mask") }
		if masks[i - 1] == mask { t.Fatal("broken test") }
	}

	// cooldown for recently accessed masks
	testInstantNanosHack += millis200

	biggerMask := newEmptyGlyphMask(12, 12)
	cache.PassMask([3]uint64{999, 999, 999}, biggerMask)
	_, found = cache.GetMask([3]uint64{0, 0, 3})
	if found { t.Fatal("expected mask to be evicted") }
	_, found = cache.GetMask([3]uint64{0, 0, 4})
	if found { t.Fatal("expected mask to be evicted") }
	_, found = cache.GetMask([3]uint64{0, 0, 1})
	if found { t.Fatal("this mask hasn't been ever passed to cache") }
	_, found = cache.GetMask([3]uint64{0, 0, 5})
	if !found { t.Fatal("expected mask to be present") }
	_, found = cache.GetMask([3]uint64{0, 0, 0})
	if !found { t.Fatal("expected mask to be present") }

	gotSize = cache.PeakSize()
	if gotSize != expectSize { t.Fatalf("expected %d, got %d", expectSize, gotSize) }
	expectSize = expectSize - int(refSize*2) + int(GlyphMaskByteSize(biggerMask))
	gotSize = cache.ApproxByteSize()
	if gotSize != expectSize { t.Fatalf("expected %d, got %d", expectSize, gotSize) }
}
