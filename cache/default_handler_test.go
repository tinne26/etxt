package cache

import "sync/atomic"
import "testing"

import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/fract"

func TestDefaultHandler(t *testing.T) {
	rast := mask.DefaultRasterizer{}
	cache := NewDefaultCache(16*1024*1024)
	handler := cache.NewHandler()
	handler.NotifyFontChange(nil)
	handler.NotifyRasterizerChange(&rast)
	handler.NotifySizeChange(12 << 6)
	handler.NotifyFractChange(fract.Point{1, 1})

	if handler.ApproxCacheByteSize() != 0 {
		t.Fatal("no mask yet size != 0")
	}

	if GlyphMaskByteSize(nil) != constMaskSizeFactor {
		t.Fatal("assumptions")
	}

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

	mask, found = cache.GetMask([3]uint64{0, 0x0000000000000000, 0x0000030000410009})
	if !found { t.Fatal("expected mask at the given key") }
	if mask != nil { t.Fatal("expected nil mask") }

	freed := cache.removeRandEntry(100000, cacheEntryInstant())
	if freed != constMaskSizeFactor {
		t.Fatalf("expected %d freed bytes, got %d", constMaskSizeFactor, freed)
	}
	atomic.AddUint32(&cache.spaceBytesLeft, constMaskSizeFactor)

	freed = cache.removeRandEntry(100000, cacheEntryInstant())
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
