//go:build gtxt

package emask

import "os"
import "log"
import "io/fs"
import "errors"
import "strings"

import "testing"

import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

var testFont *sfnt.Font
func init() { // parse test font
	workDir, err := os.Getwd()
	if err != nil { log.Fatal(err) }

	fontPath := "test_font.ttf"
	_, err = os.Stat(fontPath)
	if errors.Is(err, fs.ErrNotExist) && strings.HasSuffix(workDir, "/etxt/emask") {
		fontPath = "../test_font.ttf" // search on etxt/ instead
	}

	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) { log.Fatal(err) }
		log.Fatal("etxt requires '" + fontPath + "' file to run benchmarks")
	}
	testFont, err = sfnt.Parse(fontBytes)
	if err != nil { log.Fatal(err) }
}

func BenchmarkFauxBoldWhole(b *testing.B) {
	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetExtraWidth(1)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxBoldWhole7(b *testing.B) {
	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetExtraWidth(7)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxBoldFract(b *testing.B) {
	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetExtraWidth(0.7)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxBoldFract7(b *testing.B) {
	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetExtraWidth(7.4)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxOblique(b *testing.B) {
	contours, rasterizer, dot := getBenchMinData()
	rasterizer.SetSkewFactor(0.8) // (this is a rather extreme value)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkFauxNone(b *testing.B) {
	contours, rasterizer, dot := getBenchMinData()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Rasterize(contours, rasterizer, dot)
	}
}

func BenchmarkDefault(b *testing.B) {
	var buffer sfnt.Buffer
	size := fixed.Int26_6(72 << 6) // in pixels
	rasterizer := DefaultRasterizer{}
	_ = rasterizer.CacheSignature()
	index, err := testFont.GlyphIndex(&buffer, 'g')
	if err != nil { log.Fatal(err) }
	contours, err := testFont.LoadGlyph(&buffer, index, size, nil)
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
	index, err := testFont.GlyphIndex(&buffer, 'g')
	if err != nil { log.Fatal(err) }
	contours, err := testFont.LoadGlyph(&buffer, index, size, nil)
	if err != nil { log.Fatal(err) }
	return contours, &rasterizer, fixed.P(0, 0)
}
