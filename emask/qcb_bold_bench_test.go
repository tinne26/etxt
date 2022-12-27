//go:build bench

package emask

import "log"
import "testing"

import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

func BenchmarkFauxBoldWhole(b *testing.B) {
	if benchFont == nil { b.SkipNow() }

	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetExtraWidth(1)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxBoldWhole7(b *testing.B) {
	if benchFont == nil { b.SkipNow() }

	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetExtraWidth(7)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxBoldFract(b *testing.B) {
	if benchFont == nil { b.SkipNow() }

	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetExtraWidth(0.7)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxBoldFract7(b *testing.B) {
	if benchFont == nil { b.SkipNow() }

	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetExtraWidth(7.4)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxOblique(b *testing.B) {
	if benchFont == nil { b.SkipNow() }

	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetSkewFactor(0.8) // (this is a rather extreme value)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxNone(b *testing.B) {
	if benchFont == nil { b.SkipNow() }

	contours, rasterizer, dot := getBenchMinData()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkDefault(b *testing.B) {
	if benchFont == nil { b.SkipNow() }

	var buffer sfnt.Buffer
	size := fixed.Int26_6(72 << 6) // in pixels
	rasterizer := DefaultRasterizer{}
	_ = rasterizer.CacheSignature()
	index, err := benchFont.GlyphIndex(&buffer, 'g')
	if err != nil { log.Fatal(err) }
	contours, err := benchFont.LoadGlyph(&buffer, index, size, nil)
	if err != nil { log.Fatal(err) }
	dot := fixed.P(0, 0)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, &rasterizer, dot)
	}
}

func getBenchMinData() (sfnt.Segments, *FauxRasterizer, fixed.Point26_6) {
	var buffer sfnt.Buffer
	size := fixed.Int26_6(72 << 6) // in pixels
	rasterizer := FauxRasterizer{}
	_ = rasterizer.CacheSignature()
	index, err := benchFont.GlyphIndex(&buffer, 'g')
	if err != nil { log.Fatal(err) }
	contours, err := benchFont.LoadGlyph(&buffer, index, size, nil)
	if err != nil { log.Fatal(err) }
	return contours, &rasterizer, fixed.P(0, 0)
}
