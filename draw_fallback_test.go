//go:build gtxt

package etxt

import (
	"image"
	"testing"

	"github.com/tinne26/etxt/fract"
	"golang.org/x/image/font/sfnt"
)

// TestDrawFallbackFonts verifies that glyphs missing in the primary font are
// drawn with the font picked by the miss handler, by every draw path and by
// feeds, and that the primary font is active again afterwards.
func TestDrawFallbackFonts(t *testing.T) {
	primary, fallback := fallbackFontsForTest(t)
	segments := [][2]string{{"ab ", "primary"}, {testHanSample, "han"}, {" cd", "primary"}}
	text, expected := scriptDrawCase(segments, primary, fallback)

	renderer := NewRenderer()
	renderer.SetFont(primary)
	renderer.Utils().SetCache8MiB() // feeds require a cache handler
	renderer.Glyph().SetMissHandler(OnMissFallback(fallback))
	var drawn []drawnGlyph
	renderer.Glyph().SetDrawFunc(func(_ Target, index sfnt.GlyphIndex, origin fract.Point) {
		drawn = append(drawn, drawnGlyph{renderer.GetFont(), index, origin})
	})

	target := image.NewRGBA(image.Rect(0, 0, 512, 128))
	for _, direction := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(direction)
		for _, align := range []Align{Left, HorzCenter, Right} {
			renderer.SetAlign(align | Baseline)
			for _, wrap := range []bool{false, true} {
				drawn = drawn[:0]
				if wrap {
					renderer.DrawWithWrap(target, text, 256, 64, 512)
				} else {
					renderer.Draw(target, text, 256, 64)
				}
				if !sameGlyphs(visualLines(drawn, direction), expected) {
					t.Fatalf("%v %v (wrap %v): wrong glyphs or fonts", direction, align.Horz(), wrap)
				}
				if renderer.GetFont() != primary {
					t.Fatalf("%v %v (wrap %v): primary font not restored", direction, align.Horz(), wrap)
				}
			}
		}

		drawn = drawn[:0]
		feed := NewFeed(renderer).At(256, 64)
		for _, codePoint := range text {
			feed.Draw(target, codePoint)
		}
		if !sameGlyphs(visualLines(drawn, direction), expected) {
			t.Fatalf("%v feed: wrong glyphs or fonts", direction)
		}
		if renderer.GetFont() != primary {
			t.Fatalf("%v feed: primary font not restored", direction)
		}
	}
}
