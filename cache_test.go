//go:build gtxt && test

package etxt

import "image"
import "testing"

func TestDefaultCacheConsistency(t *testing.T) {
	// this test consists in writing some text without caching
	// while changing text positioning and fonts, and then doing
	// the same with caching to ensure that the results are the
	// same
	targetNoCache  := image.NewRGBA(image.Rect(0, 0, 256, 128))
	targetYesCache := image.NewRGBA(image.Rect(0, 0, 256, 128))
	cache := NewDefaultCache(8*1024*1024) // 8MiB cache
	cacheHandler := cache.NewHandler()
	renderer := NewStdRenderer()
	renderer.SetAlign(YCenter, XCenter)

	var drawProc = func() {
		y := 16
		renderer.SetFont(testFont)
		renderer.Draw("education", 128, y)
		y += 24
		renderer.SetFont(testFont2)
		renderer.Draw("failure", 128, y)
		y += 24
		renderer.SetFont(testFont)
		renderer.Draw("programming", 128, y)
		y += 24
		renderer.SetFont(testFont2)
		renderer.Draw("disaster", 128, y)
	}

	// first test config
	renderer.SetSizePx(16)
	renderer.SetQuantizerStep(64, 64)

	renderer.SetCacheHandler(nil)
	renderer.SetTarget(targetNoCache)
	drawProc()
	renderer.SetCacheHandler(cacheHandler)
	renderer.SetTarget(targetYesCache)
	drawProc()
	compareDrawResults("FullQuant16", t, targetNoCache, targetYesCache)

	// second test config
	renderer.SetSizePx(12)
	renderer.SetQuantizerStep(1, 64)

	renderer.SetCacheHandler(nil)
	renderer.SetTarget(targetNoCache)
	drawProc()
	renderer.SetCacheHandler(cacheHandler)
	renderer.SetTarget(targetYesCache)
	drawProc()
	compareDrawResults("VertQuant12", t, targetNoCache, targetYesCache)

	// third test config
	renderer.SetSizePx(17)
	renderer.SetQuantizerStep(1, 1)

	renderer.SetCacheHandler(nil)
	renderer.SetTarget(targetNoCache)
	drawProc()
	renderer.SetCacheHandler(cacheHandler)
	renderer.SetTarget(targetYesCache)
	drawProc()
	compareDrawResults("NoQuant17", t, targetNoCache, targetYesCache)

	// ...
}

func compareDrawResults(testKey string, t *testing.T, targetNoCache, targetYesCache *image.RGBA) {
	const ShowMeVisually = false
	for i := 0; i < len(targetNoCache.Pix); i++ {
		a := targetNoCache.Pix[i]
		b := targetYesCache.Pix[i]
		if a != b {
			if ShowMeVisually {
				debugExport("test_failure_no_cache.png", targetNoCache)
				debugExport("test_failure_yes_cache.png", targetYesCache)
			}
			t.Fatalf("Cache consistency test '%s' failed, i = %d has values %d vs %d", testKey, i, a, b)
		}
	}
}
