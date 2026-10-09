//go:build gtxt

package etxt

import (
	"image"
	"sort"
	"testing"
	"unicode"

	"github.com/tinne26/etxt/fract"
	"golang.org/x/image/font/sfnt"
)

type drawnGlyph struct {
	font   *sfnt.Font
	index  sfnt.GlyphIndex
	origin fract.Point
}

// scriptDrawCase applies the given fonts to the split test strings, and
// returns the text and the glyphs expected for it, line by line in text order.
func scriptDrawCase(segments [][2]string, primary, han *sfnt.Font) (string, [][]drawnGlyph) {
	lines := [][]drawnGlyph{nil}
	for _, segment := range segments {
		font := segmentFont(segment[1], primary, han)
		if font == nil { // line break
			lines = append(lines, nil)
			continue
		}
		for _, codePoint := range segment[0] {
			index, _ := font.GlyphIndex(nil, codePoint)
			lines[len(lines)-1] = append(lines[len(lines)-1], drawnGlyph{font: font, index: index})
		}
	}
	return scriptCaseText(segments), lines
}

// newScriptDrawRenderer returns a renderer with a script font for Han and a
// custom draw function that records each glyph it draws.
func newScriptDrawRenderer(t *testing.T) (renderer *Renderer, drawn *[]drawnGlyph, primary, han *sfnt.Font) {
	t.Helper()
	primary, han = scriptFontsForTest(t)
	renderer = NewRenderer()
	renderer.SetFont(primary)
	renderer.Script().SetFont(unicode.Han, han)
	drawn = &[]drawnGlyph{}
	renderer.Glyph().SetDrawFunc(func(_ Target, index sfnt.GlyphIndex, origin fract.Point) {
		*drawn = append(*drawn, drawnGlyph{renderer.GetFont(), index, origin})
	})
	return renderer, drawn, primary, han
}

// TestDrawScriptFonts verifies that every combination of direction, horizontal
// align and wrapping draws each glyph with the expected font, and at the same
// relative position. These combinations use different draw paths: some visit
// each line backwards, and RTL places glyphs leftwards. Fonts depend only on
// the position of each rune in the text, so the order in which a path visits
// the glyphs must not change the result.
func TestDrawScriptFonts(t *testing.T) {
	renderer, drawn, primary, han := newScriptDrawRenderer(t)
	renderer.Fract().SetHorzQuantization(QtNone) // see sameSpacing
	target := image.NewRGBA(image.Rect(0, 0, 512, 128))
	for _, segments := range scriptCases {
		text, expected := scriptDrawCase(segments, primary, han)
		for _, direction := range []Direction{LeftToRight, RightToLeft} {
			renderer.SetDirection(direction)

			// A negative limit draws without wrapping. The text width rounded up is
			// the narrowest limit that doesn't wrap any line.
			widthLimits := []int{-1, renderer.Measure(text).Width().ToIntCeil()}
			var reference [][]drawnGlyph // left aligned Draw
			for _, align := range []Align{Left, HorzCenter, Right} {
				renderer.SetAlign(align | VertCenter) // makes DrawWithWrap() measure the text before drawing it
				for _, widthLimit := range widthLimits {
					*drawn = (*drawn)[:0]
					if widthLimit < 0 {
						renderer.Draw(target, text, 256, 64)
					} else {
						renderer.DrawWithWrap(target, text, 256, 64, widthLimit)
					}

					lines := visualLines(*drawn, direction)
					if !sameGlyphs(lines, expected) {
						t.Fatalf("%q %v %v (width limit %d): wrong glyphs or fonts", text, direction, align.Horz(), widthLimit)
					}
					if reference == nil {
						reference = lines
					} else if !sameSpacing(lines, reference) {
						t.Fatalf("%q %v %v (width limit %d): spacing differs from left aligned Draw", text, direction, align.Horz(), widthLimit)
					}
					if renderer.GetFont() != primary {
						t.Fatalf("%q %v %v (width limit %d): primary font not restored", text, direction, align.Horz(), widthLimit)
					}
				}
			}
		}
	}
}

// visualLines splits drawn glyphs into lines where their Y changes, and sorts
// each line in text order: left to right for LTR, right to left for RTL.
// All draw paths draw lines from top to bottom, only the order within each
// line differs.
func visualLines(drawn []drawnGlyph, direction Direction) [][]drawnGlyph {
	var lines [][]drawnGlyph
	for i, glyph := range drawn {
		if i == 0 || glyph.origin.Y != drawn[i-1].origin.Y {
			lines = append(lines, nil)
		}
		lines[len(lines)-1] = append(lines[len(lines)-1], glyph)
	}
	for _, line := range lines {
		sort.Slice(line, func(i, j int) bool {
			if direction == RightToLeft {
				return line[i].origin.X > line[j].origin.X
			}
			return line[i].origin.X < line[j].origin.X
		})
	}
	return lines
}

// sameGlyphs reports whether both layouts have the same glyphs and fonts,
// ignoring positions.
func sameGlyphs(lines, expected [][]drawnGlyph) bool {
	if len(lines) != len(expected) {
		return false
	}
	for i := range lines {
		if len(lines[i]) != len(expected[i]) {
			return false
		}
		for j := range lines[i] {
			if lines[i][j].font != expected[i][j].font || lines[i][j].index != expected[i][j].index {
				return false
			}
		}
	}
	return true
}

// sameSpacing reports whether the glyphs of both layouts are equally spaced
// within each line. Lines may be shifted as a whole. Expects layouts with the
// same glyphs, drawn without horizontal quantization: centered lines can start
// at fractional positions, and rounding glyphs from a different start can
// change the spacing between them by a pixel.
func sameSpacing(lines, reference [][]drawnGlyph) bool {
	for i := range lines {
		shift := lines[i][0].origin.X - reference[i][0].origin.X
		for j := range lines[i] {
			if lines[i][j].origin.X-shift != reference[i][j].origin.X {
				return false
			}
		}
	}
	return true
}

// TestDrawScriptFontsWrapping verifies that wrapping doesn't change fonts:
// "." follows "a" in the text, so it keeps the primary font even when it
// starts a wrapped line next to "中".
func TestDrawScriptFontsWrapping(t *testing.T) {
	renderer, drawn, primary, han := newScriptDrawRenderer(t)
	glyphA, _ := primary.GlyphIndex(nil, 'a')
	glyphDot, _ := primary.GlyphIndex(nil, '.')
	glyphHan, _ := han.GlyphIndex(nil, '中')
	expected := [][]drawnGlyph{
		{{font: primary, index: glyphA}},
		{{font: primary, index: glyphDot}, {font: han, index: glyphHan}},
	}

	target := image.NewRGBA(image.Rect(0, 0, 512, 128))
	for _, direction := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(direction)
		widthLimit := renderer.Measure("a .中").Width().ToIntCeil() - 1
		*drawn = (*drawn)[:0]
		renderer.DrawWithWrap(target, "a .中", 256, 32, widthLimit)
		if !sameGlyphs(visualLines(*drawn, direction), expected) {
			t.Fatalf("%v: expected \"a\", then \".\" with the primary font and \"中\" with the Han font", direction)
		}
	}
}

// TestDrawScriptFontsLineMetrics verifies that line metrics come from the
// primary font, even when a line ends with a glyph from a script font, with
// and without wrapping.
func TestDrawScriptFontsLineMetrics(t *testing.T) {
	renderer, _, primary, _ := newScriptDrawRenderer(t)
	metrics := &lineMetricRecorder{}
	renderer.SetSizer(metrics)
	target := image.NewRGBA(image.Rect(0, 0, 512, 128))
	for _, direction := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(direction)
		renderer.Draw(target, lineMetricsText, 8, 32)
		renderer.DrawWithWrap(target, lineMetricsText, 8, 32, lineMetricsWrapLimit(t, renderer))
	}
	if !metrics.recordedOnly(primary) {
		t.Fatalf("line metrics used a font other than the primary: %v", metrics.fonts)
	}
}

// TestDrawScriptFontsAllocations verifies that drawing text that switches
// between script fonts doesn't allocate once the renderer has drawn it before.
// The first time, the buffer for font switches may grow.
func TestDrawScriptFontsAllocations(t *testing.T) {
	primary, han := scriptFontsForTest(t)

	renderer := NewRenderer()
	renderer.SetFont(primary)
	renderer.Script().SetFont(unicode.Han, han)
	renderer.Glyph().SetDrawFunc(func(Target, sfnt.GlyphIndex, fract.Point) {})
	target := image.NewRGBA(image.Rect(0, 0, 512, 128))
	text := scriptCaseText(scriptCases[0])
	forceWrapLimit := renderer.Measure(text).IntWidth() / 2 // wraps the first line

	allocs := testing.AllocsPerRun(100, func() {
		renderer.Draw(target, text, 256, 32)
		renderer.DrawWithWrap(target, text, 256, 32, forceWrapLimit)
	})
	if allocs != 0 {
		t.Fatalf("expected no allocations, got %v", allocs)
	}
}
