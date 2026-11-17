package etxt

import (
	"testing"

	"github.com/tinne26/etxt/internal/testfont"
	"golang.org/x/image/font/sfnt"
)

// fallbackFontsForTest returns a primary font with Latin glyphs and without
// Han glyphs, and a fallback font with Han glyphs.
func fallbackFontsForTest(tb testing.TB) (primary, fallback *sfnt.Font) {
	tb.Helper()
	fonts := testFonts.Matching(tb,
		testfont.Filter{Has: testLatinSample, Lacks: testHanSample},
		testfont.Filter{Has: testHanSample},
	)
	return fonts[0].Font, fonts[1].Font
}

// TestMeasureFallbackFonts verifies that glyphs missing in the primary font
// are measured with the font picked by the miss handler, without kerning
// across fonts, and without allocations once the text was measured before.
func TestMeasureFallbackFonts(t *testing.T) {
	primary, fallback := fallbackFontsForTest(t)
	segments := [][2]string{{"ab ", "primary"}, {testHanSample, "han"}, {" cd", "primary"}}
	text := scriptCaseText(segments)

	renderer := NewRenderer()
	renderer.SetFont(primary)
	renderer.Glyph().SetMissHandler(OnMissFallback(fallback))
	renderer.Fract().SetHorzQuantization(QtNone) // see scriptCaseWidth
	for _, direction := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(direction)
		expected := scriptCaseWidth(segments, primary, fallback, direction)
		if width := renderer.Measure(text).Width(); width != expected {
			t.Fatalf("%v: expected width %v, got %v", direction, expected, width)
		}
		if width := renderer.MeasureWithWrap(text, expected.ToIntCeil()).Width(); width != expected {
			t.Fatalf("%v (wrap): expected width %v, got %v", direction, expected, width)
		}
		if renderer.GetFont() != primary {
			t.Fatalf("%v: primary font not restored", direction)
		}
	}

	allocs := testing.AllocsPerRun(100, func() { renderer.Measure(text) })
	if allocs != 0 {
		t.Fatalf("expected no allocations, got %v", allocs)
	}
}
