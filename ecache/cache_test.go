//go:build gtxt

package ecache

import "time"
import "image"
import "sync/atomic"

import "testing"

import "github.com/tinne26/etxt/emask"

import "golang.org/x/image/math/fixed"

func TestDefaultCache(t *testing.T) {
	masks := make([]GlyphMask, 10)

	rect := image.Rect(0, 0, 10, 10)
	for i := 0; i < 10; i++ {
		masks[i] = GlyphMask(image.NewAlpha(rect))
		masks[i].Pix[i] = 1
	}
	refSize := GlyphMaskByteSize(masks[0])

	cache, err := NewDefaultCache(int(refSize*8))
	if err != nil { panic(err) }

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
	if mask.Pix[2] != 1 { t.Fatal("absurd test") }

	for i := 3; i < 10; i++ {
		if i <= 5 { time.Sleep(200*time.Millisecond) } // keep mask additions appart
		cache.PassMask([3]uint64{0, 0, uint64(i)}, masks[i])
	}

	for i := 3; i < 10; i++ {
		mask, found = cache.GetMask([3]uint64{0, 0, uint64(i)})
		if !found { t.Fatal("expected to find mask") }
		if mask.Pix[i - 1] != 0 || mask.Pix[i] != 1 { t.Fatal("wrong mask") }
	}

	gotSize = cache.ApproxByteSize()
	expectSize := int(refSize)*8
	if gotSize != expectSize { t.Fatalf("expected %d, got %d", expectSize, gotSize) }
	gotSize = cache.PeakSize()
	if gotSize != expectSize { t.Fatalf("expected %d, got %d", expectSize, gotSize) }

	time.Sleep(200*time.Millisecond)
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
		if mask.Pix[i - 1] != 0 || mask.Pix[i] != 1 { t.Fatal("wrong mask") }
	}

	// cooldown for recently accessed masks
	time.Sleep(200*time.Millisecond)

	biggerMask := GlyphMask(image.NewAlpha(image.Rect(0, 0, 12, 12)))
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

func TestDefaultHandler(t *testing.T) {
	cache, err := NewDefaultCache(16*1024*1024)
	if err != nil { panic(err) }

	rast := emask.DefaultRasterizer{}
	rast.SetHighByte(0x77)

	handler := cache.NewHandler()
	handler.NotifyFontChange(nil)
	handler.NotifyRasterizerChange(&rast)
	handler.NotifySizeChange(12 << 6)
	handler.NotifyFractChange(fixed.Point26_6{1, 1})

	if handler.ApproxCacheByteSize() != 0 {
		t.Fatal("no mask yet size != 0")
	}

	if GlyphMaskByteSize(nil) != constMaskSizeFactor { t.Fatal("assumptions") }

	_, found := handler.GetMask(9)
	if found { t.Fatal("no mask in the cache") }
	handler.PassMask(9, nil)
	mask, found := handler.GetMask(9)
	if !found { t.Fatal("expected mask in cache") }
	if mask != nil { t.Fatal("expected nil mask") }

	gotSize := handler.PeakCacheSize()
	if gotSize != constMaskSizeFactor {
		t.Fatalf("expected %d bytes, got %d", constMaskSizeFactor, gotSize)
	}

	mask, found = cache.GetMask([3]uint64{0, 0x7700000000000000, 0x0000030000410009})
	if !found { t.Fatal("expected mask at the given key") }
	if mask != nil { t.Fatal("expected nil mask") }

	freed := cache.removeRandEntry(100000, CacheEntryInstant())
	if freed != constMaskSizeFactor {
		t.Fatalf("expected %d freed bytes, got %d", constMaskSizeFactor, freed)
	}
	atomic.AddUint32(&cache.spaceBytesLeft, constMaskSizeFactor)

	freed = cache.removeRandEntry(100000, CacheEntryInstant())
	if freed != 0 {
		t.Fatalf("expected 0 freed bytes, got %d", freed)
	}

	gotSize = handler.ApproxCacheByteSize()
	if gotSize != 0 { t.Fatalf("expected 0 bytes, got %d", gotSize) }

	gotSize = handler.PeakCacheSize()
	if gotSize != constMaskSizeFactor {
		t.Fatalf("expected %d bytes, got %d", constMaskSizeFactor, gotSize)
	}
}
