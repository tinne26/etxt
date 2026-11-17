package etxt

import (
	"image/color"
	"testing"
	"unicode"

	"github.com/tinne26/etxt/cache"
	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/etxt/mask"
	"github.com/tinne26/etxt/sizer"
	"golang.org/x/image/font/sfnt"
)

// BlendMode is left out: it's ebiten.Blend or uint8 depending on the
// build tags, with no non-default value available to both.
func TestStoreRestoreState(t *testing.T) {
	pair := testFontsWithRunes(t, "", "") // any two different fonts
	fontA, fontB := pair[0], pair[1]

	check := func(ok bool, failure string) {
		t.Helper()
		if !ok {
			t.Fatal(failure)
		}
	}

	sizerA, sizerB := &sizer.DefaultSizer{}, &sizer.PaddedKernSizer{}
	rasterizerA, rasterizerB := &mask.DefaultRasterizer{}, &mask.SharpRasterizer{}
	colorA, colorB := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}

	renderer := NewRenderer()
	renderer.SetFont(fontA)
	renderer.Script().SetFont(unicode.Han, fontA)
	renderer.SetAlign(Left | Baseline)
	renderer.SetColor(colorA)
	renderer.SetSize(16)
	renderer.SetScale(1)
	renderer.SetSizer(sizerA)
	renderer.Glyph().SetRasterizer(rasterizerA)
	renderer.Fract().SetHorzQuantization(Qt4th)
	renderer.Fract().SetVertQuantization(QtFull)
	renderer.SetDirection(LeftToRight)

	renderer.Utils().StoreState()

	renderer.SetFont(fontB)
	renderer.Script().SetFont(unicode.Han, fontB)
	renderer.SetAlign(Right | Top)
	renderer.SetColor(colorB)
	renderer.SetSize(32)
	renderer.SetScale(2)
	renderer.SetSizer(sizerB)
	renderer.Glyph().SetRasterizer(rasterizerB)
	renderer.Fract().SetHorzQuantization(QtNone)
	renderer.Fract().SetVertQuantization(QtHalf)
	renderer.SetDirection(RightToLeft)

	cacheHandler := cache.NewDefaultCache(1024 * 1024).NewHandler()
	renderer.SetCacheHandler(cacheHandler)
	renderer.Glyph().SetDrawFunc(func(Target, sfnt.GlyphIndex, fract.Point) {})
	renderer.Glyph().SetLineChangeFunc(func(LineChangeDetails) {})
	renderer.Glyph().SetMissHandler(OnMissNotdef)

	check(renderer.Utils().RestoreState(), "expected a stored state to restore")

	// stored properties are back to their stored values
	check(renderer.GetFont() == fontA, "font not restored")
	check(renderer.Script().GetFont(unicode.Han) == fontA, "script fonts not restored")
	check(renderer.GetAlign() == Left|Baseline, "align not restored")
	check(renderer.GetColor() == colorA, "color not restored")
	check(renderer.GetSize() == 16 && renderer.GetScale() == 1, "size and scale not restored")
	check(renderer.GetSizer() == sizerA, "sizer not restored")
	check(renderer.Glyph().GetRasterizer() == rasterizerA, "rasterizer not restored")
	horz, vert := renderer.Fract().GetQuantization()
	check(horz == Qt4th && vert == QtFull, "quantization not restored")
	check(renderer.GetDirection() == LeftToRight, "direction not restored")

	// everything else keeps its latest value
	check(renderer.GetCacheHandler() == cacheHandler, "cache handler is not part of the state")
	check(renderer.customDrawFn != nil, "draw func is not part of the state")
	check(renderer.lineChangeFn != nil, "line change func is not part of the state")
	check(renderer.missHandlerFn != nil, "miss handler is not part of the state")

	// script fonts are shared with stored states until written, so
	// nested levels must not leak replacements, insertions, removals
	// or clears into each other, nor into later visits of a level
	expectScripts := func(context string, han, greek, cyrillic *sfnt.Font) {
		t.Helper()
		if renderer.Script().GetFont(unicode.Han) != han ||
			renderer.Script().GetFont(unicode.Greek) != greek ||
			renderer.Script().GetFont(unicode.Cyrillic) != cyrillic {
			t.Fatalf("unexpected script fonts %s", context)
		}
	}

	renderer.Utils().StoreState()
	renderer.Script().SetFont(unicode.Han, fontB)      // replace
	renderer.Script().SetFont(unicode.Cyrillic, fontA) // insert before Han
	renderer.Utils().StoreState()
	renderer.Script().SetFont(unicode.Cyrillic, nil) // remove
	renderer.Script().SetFont(unicode.Greek, fontB)  // insert between
	renderer.Utils().StoreState()
	renderer.Script().Clear()
	expectScripts("after clearing", nil, nil, nil)

	renderer.Utils().RestoreState()
	expectScripts("on the third level", fontB, fontB, nil)
	renderer.Utils().RestoreState()
	expectScripts("on the second level", fontB, nil, fontA)
	renderer.Utils().RestoreState()
	expectScripts("on the first level", fontA, nil, nil)

	renderer.Utils().StoreState()
	renderer.Script().SetFont(unicode.Greek, fontA)
	expectScripts("on a second visit to the second level", fontA, fontA, nil)
	renderer.Utils().RestoreState()
	expectScripts("back on the first level", fontA, nil, nil)

	renderer.Utils().StoreState() // a level without writes
	renderer.Utils().StoreState()
	renderer.Script().SetFont(unicode.Greek, fontB)
	renderer.Utils().RestoreState()
	renderer.Utils().RestoreState()
	expectScripts("after restoring a level without writes", fontA, nil, nil)

	// each depth keeps its memory across visits, including visits without
	// writes, so once every depth has been written to, nothing allocates
	allocs := testing.AllocsPerRun(100, func() {
		renderer.Utils().StoreState() // writes on both depths
		renderer.Script().SetFont(unicode.Han, fontB)
		renderer.Utils().StoreState()
		renderer.Script().SetFont(unicode.Greek, fontB)
		renderer.Utils().RestoreState()
		renderer.Utils().RestoreState()

		renderer.Utils().StoreState() // no writes on either depth
		renderer.Utils().StoreState()
		renderer.Utils().RestoreState()
		renderer.Utils().RestoreState()
	})
	if allocs != 0 {
		t.Fatalf("expected no allocations for temporary script font changes, got %v", allocs)
	}
	expectScripts("after the temporary changes", fontA, nil, nil)
}

// TestZeroValueRenderer verifies that a zero value renderer can be set up
// with FillMissingProperties() and then used like any other.
func TestZeroValueRenderer(t *testing.T) {
	font := testFontsWithRunes(t, "abc")[0]

	var renderer Renderer
	renderer.Utils().AssertMaxStoredStates(0)
	if renderer.Utils().RestoreState() {
		t.Fatal("expected no state to restore")
	}
	renderer.SetFont(font)
	renderer.Utils().FillMissingProperties()

	renderer.Utils().StoreState()
	renderer.SetSize(32)
	if !renderer.Utils().RestoreState() || renderer.GetSize() != 16 {
		t.Fatal("expected the stored state to be restored")
	}
	if renderer.Measure("abc").Width() == 0 {
		t.Fatal("expected a non-zero width")
	}
	drawn := 0
	renderer.Glyph().SetDrawFunc(func(Target, sfnt.GlyphIndex, fract.Point) { drawn += 1 })
	renderer.Draw(nil, "abc", 0, 16)
	if drawn != 3 {
		t.Fatalf("expected 3 glyphs drawn, got %d", drawn)
	}
}
