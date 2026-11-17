package etxt

import (
	"math/rand"
	"testing"
	"time"

	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/etxt/sizer"
	"golang.org/x/image/font/sfnt"
)

// TestFeedVsDraw verifies that Feed and Renderer.Draw place glyphs the same.
func TestFeedVsDraw(t *testing.T) {
	type drawnGlyph struct {
		index  sfnt.GlyphIndex
		origin fract.Point
	}

	const text = "AVATAR To\n\nVoyage, xyz!"
	seed := time.Now().UnixNano() % 100000
	rng := rand.New(rand.NewSource(seed))

	renderer := NewRenderer()
	renderer.SetFont(testFontsWithRunes(t, text)[0])
	renderer.Utils().FillMissingProperties()
	renderer.SetAlign(Left | Baseline) // feeds draw from the pen position, as Draw does with this align
	var drawn []drawnGlyph
	renderer.Glyph().SetDrawFunc(func(_ Target, index sfnt.GlyphIndex, origin fract.Point) {
		drawn = append(drawn, drawnGlyph{index, origin})
	})

	expectFeedMatchesDraw := func(config string) {
		t.Helper()
		x, y := rng.Intn(512), rng.Intn(512)

		drawn = drawn[:0]
		renderer.Draw(nil, text, x, y)
		want := append([]drawnGlyph(nil), drawn...)

		drawn = drawn[:0]
		feed := NewFeed(renderer).At(x, y)
		for _, codePoint := range text {
			if codePoint == '\n' {
				feed.LineBreak()
			} else {
				feed.Draw(nil, codePoint)
			}
		}

		if len(drawn) != len(want) {
			t.Fatalf("%s (seed %d): feed drew %d glyphs, Draw drew %d", config, seed, len(drawn), len(want))
		}
		for i := range want {
			if drawn[i] != want[i] {
				t.Fatalf("%s (seed %d): glyph %d: feed drew %v, Draw drew %v", config, seed, i, drawn[i], want[i])
			}
		}
	}

	expectFeedMatchesDraw("without cache")
	renderer.Utils().SetCache8MiB()
	expectFeedMatchesDraw("with cache")
	renderer.SetSize(17.3)
	expectFeedMatchesDraw("fractional size")
	renderer.Fract().SetHorzQuantization(QtNone)
	renderer.Fract().SetVertQuantization(QtNone)
	expectFeedMatchesDraw("no quantization")
	var paddedSizer sizer.PaddedKernSizer
	paddedSizer.SetPadding(fract.FromFloat64(2.5))
	renderer.SetSizer(&paddedSizer)
	expectFeedMatchesDraw("padded kerning")
	renderer.SetSizer(&growingLineAdvanceSizer{})
	expectFeedMatchesDraw("line advance by consecutive line breaks")
}

// growingLineAdvanceSizer advances further on each consecutive line break,
// so lineBreakNth affects the line advance, for testing.
type growingLineAdvanceSizer struct{ sizer.DefaultSizer }

func (self *growingLineAdvanceSizer) LineAdvance(font *sfnt.Font, buffer *sfnt.Buffer, size fract.Unit, lineBreakNth int) fract.Unit {
	return self.DefaultSizer.LineAdvance(font, buffer, size, lineBreakNth) + fract.Unit(lineBreakNth)*fract.One
}
