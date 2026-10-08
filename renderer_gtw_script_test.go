package etxt

import (
	"strings"
	"testing"
	"unicode"

	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/etxt/sizer"
	"golang.org/x/image/font/sfnt"
)

// samples for picking test fonts by glyph coverage
const testLatinSample = "aZ09 .,:;!?()'\""
const testHanSample = "中文你是一"

// scriptFontsForTest returns two different test fonts: a primary font with
// Latin glyphs, and a Han font that also has Latin glyphs, because punctuation
// next to Han text uses the Han font.
func scriptFontsForTest(tb testing.TB) (primary, han *sfnt.Font) {
	tb.Helper()
	fonts := testFontsWithRunes(tb, testLatinSample, testHanSample+testLatinSample)
	return fonts[0], fonts[1]
}

func TestScriptFontAssignment(t *testing.T) {
	pair := testFontsWithRunes(t, "", "") // any two different fonts
	fontA, fontB := pair[0], pair[1]
	renderer := NewRenderer()
	renderer.SetFont(fontA)

	if renderer.Script().GetFont(unicode.Han) != nil {
		t.Fatal("expected no font for an unassigned script")
	}

	renderer.Script().SetFont(unicode.Han, fontB)
	if renderer.Script().GetFont(unicode.Han) != fontB {
		t.Fatal("expected fontB for Han")
	}
	if renderer.GetFont() != fontA {
		t.Fatal("assigning a script font must not change the active font")
	}

	// a script can be reassigned, and one font can serve many scripts
	renderer.Script().SetFont(unicode.Han, fontA)
	renderer.Script().SetFont(unicode.Cyrillic, fontA)
	if renderer.Script().GetFont(unicode.Han) != fontA {
		t.Fatal("expected Han reassignment to fontA")
	}
	if renderer.Script().GetFont(unicode.Cyrillic) != fontA {
		t.Fatal("expected fontA for Cyrillic")
	}

	// a nil font removes the assignment, and only that one
	renderer.Script().SetFont(unicode.Han, nil)
	if renderer.Script().GetFont(unicode.Han) != nil {
		t.Fatal("expected the Han assignment to be removed")
	}
	if renderer.Script().GetFont(unicode.Cyrillic) != fontA {
		t.Fatal("expected the Cyrillic assignment to survive")
	}

	renderer.Script().Clear()
	if renderer.Script().GetFont(unicode.Cyrillic) != nil {
		t.Fatal("expected Clear() to remove every assignment")
	}
}

func TestScriptEach(t *testing.T) {
	pair := testFontsWithRunes(t, "", "") // any two different fonts
	fontA, fontB := pair[0], pair[1]
	renderer := NewRenderer()
	renderer.Script().SetFont(unicode.Han, fontA)
	renderer.Script().SetFont(unicode.Greek, fontB)
	renderer.Script().SetFont(unicode.Cyrillic, fontA)

	var names []string
	var scripts []*unicode.RangeTable
	var fonts []*sfnt.Font
	renderer.Script().Each(func(name string, script *unicode.RangeTable, font *sfnt.Font) bool {
		names = append(names, name)
		scripts = append(scripts, script)
		fonts = append(fonts, font)
		return true
	})

	expectedNames := []string{"Cyrillic", "Greek", "Han"}
	expectedScripts := []*unicode.RangeTable{unicode.Cyrillic, unicode.Greek, unicode.Han}
	expectedFonts := []*sfnt.Font{fontA, fontB, fontA}
	if len(names) != len(expectedNames) {
		t.Fatalf("expected %d assignments, got %d", len(expectedNames), len(names))
	}
	for i := range expectedNames {
		if names[i] != expectedNames[i] || scripts[i] != expectedScripts[i] || fonts[i] != expectedFonts[i] {
			t.Fatalf("unexpected assignment at index %d: %s", i, names[i])
		}
	}

	count := 0
	renderer.Script().Each(func(string, *unicode.RangeTable, *sfnt.Font) bool {
		count += 1
		return false
	})
	if count != 1 {
		t.Fatalf("expected Each() to stop on false, ran %d times", count)
	}
}

func TestScriptInvalidScriptPanics(t *testing.T) {
	invalid := map[string]*unicode.RangeTable{
		"nil":         nil,
		"Ideographic": unicode.Ideographic, // a property, not a script
	}
	for name, script := range invalid {
		// nil fonts remove assignments, but scripts are validated first
		if doesNotPanic(func() { NewRenderer().Script().SetFont(script, nil) }) {
			t.Fatalf("expected the %s script to panic", name)
		}
	}
}

// scriptCases defines test strings split into segments by the script / font
// they must use. Contextual code points like punctuation typically follow the
// preceding script (or after them if at the start of the line). Consecutive
// segments always use different fonts.
var scriptCases = [][][2]string{
	{
		{`"中文',`, "han"}, // the leading quote starts the line, so it takes the font after it
		{`a1;`, "primary"},
		{`中`, "han"},
		{"\n", ""},
		{`(c):`, "primary"},
		{`你`, "han"},
	},
	{
		{`中文`, "han"},
		{"\n", ""},
		{`12 .`, "primary"}, // no letters on the line: primary font
	},
}

func segmentFont(name string, primary, han *sfnt.Font) *sfnt.Font {
	switch name {
	case "primary":
		return primary
	case "han":
		return han
	case "":
		return nil
	default:
		panic("unknown segment font " + name)
	}
}

// scriptCaseText joins the text of the given segments.
func scriptCaseText(segments [][2]string) string {
	var text strings.Builder
	for _, segment := range segments {
		text.WriteString(segment[0])
	}
	return text.String()
}

// scriptCaseWidth returns the width of the widest line in the given segments.
// It measures each segment alone, with the segment's font as the primary one,
// and adds up the widths of each line. This works because kerning stops at
// font changes. Kerning depends on which glyph is on the left, and RTL lays
// out text from right to left, so a line can be wider or narrower in RTL than
// in LTR. Segments must be measured in the same direction as the full text.
// Widths are unquantized, because rounding each segment separately doesn't
// match rounding a whole line.
func scriptCaseWidth(segments [][2]string, primary, han *sfnt.Font, direction Direction) fract.Unit {
	renderer := NewRenderer()
	renderer.Fract().SetHorzQuantization(QtNone)
	renderer.SetDirection(direction)
	var width, lineWidth fract.Unit
	for _, segment := range segments {
		font := segmentFont(segment[1], primary, han)
		if font == nil { // line break
			lineWidth = 0
			continue
		}
		renderer.SetFont(font)
		lineWidth += renderer.Measure(segment[0]).Width()
		if lineWidth > width {
			width = lineWidth
		}
	}
	return width
}

// TestMeasureScriptFonts verifies that each rune is measured with the font of
// its segment in scriptCases. The expected width of each line is the sum of
// its segments, each one measured alone with its own font. Measure() and
// MeasureWithWrap() have separate code for LTR and RTL, so all four are
// checked.
func TestMeasureScriptFonts(t *testing.T) {
	primary, han := scriptFontsForTest(t)
	renderer := NewRenderer()
	renderer.SetFont(primary)
	renderer.Script().SetFont(unicode.Han, han)
	renderer.Fract().SetHorzQuantization(QtNone) // see scriptCaseWidth
	for _, segments := range scriptCases {
		text := scriptCaseText(segments)
		for _, direction := range []Direction{LeftToRight, RightToLeft} {
			renderer.SetDirection(direction)
			expected := scriptCaseWidth(segments, primary, han, direction)

			// A negative limit measures without wrapping. The text width rounded up
			// is the narrowest limit that doesn't wrap any line.
			for _, widthLimit := range []int{-1, expected.ToIntCeil()} {
				var rect fract.Rect
				if widthLimit < 0 {
					rect = renderer.Measure(text)
				} else {
					rect = renderer.MeasureWithWrap(text, widthLimit)
				}
				if rect.Width() != expected {
					t.Fatalf("%q %v (width limit %d): expected width %v, got %v", text, direction, widthLimit, expected, rect.Width())
				}
				if renderer.GetFont() != primary {
					t.Fatalf("%q %v (width limit %d): primary font not restored", text, direction, widthLimit)
				}
			}
		}
	}
}

// TestMeasureScriptFontsWrapping verifies that wrapping doesn't change fonts.
// "a .中" wraps after the space. The "." follows "a" in the text, so it keeps
// the primary font even though it starts the second line, next to "中".
func TestMeasureScriptFontsWrapping(t *testing.T) {
	primary, han := scriptFontsForTest(t)
	renderer := NewRenderer()
	renderer.SetFont(primary)
	renderer.Script().SetFont(unicode.Han, han)
	renderer.Fract().SetHorzQuantization(QtNone) // see scriptCaseWidth
	wrapped := [][2]string{{"a", "primary"}, {"\n", ""}, {".", "primary"}, {"中", "han"}}
	for _, direction := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(direction)
		widthLimit := renderer.Measure("a .中").Width().ToIntCeil() - 1
		expected := scriptCaseWidth(wrapped, primary, han, direction)
		if width := renderer.MeasureWithWrap("a .中", widthLimit).Width(); width != expected {
			t.Fatalf("%v: expected width %v, got %v", direction, expected, width)
		}
	}
}

// lineMetricsText has lines that end with a Han glyph.
const lineMetricsText = "ab 中文\n你"

// lineMetricsWrapLimit returns a width limit that wraps lineMetricsText into
// "ab", "中文" and "你". To find that wrap point, the renderer measures "中"
// first, so the Han font is active when the renderer advances to the next line.
func lineMetricsWrapLimit(t *testing.T, renderer *Renderer) int {
	t.Helper()
	widthLimit := renderer.Measure("ab ").IntWidth()
	if hanWidth := renderer.Measure("中文").IntWidth(); hanWidth > widthLimit {
		widthLimit = hanWidth
	}
	if renderer.Measure("ab 中").Width() <= fract.FromInt(widthLimit) {
		t.Skip("the test fonts don't wrap lineMetricsText after \"ab\"")
	}
	return widthLimit
}

// lineMetricRecorder is a sizer. Each time the renderer asks it for a line
// height or a line advance, it records the font.
type lineMetricRecorder struct {
	sizer.DefaultSizer
	fonts []*sfnt.Font
}

func (self *lineMetricRecorder) LineHeight(font *sfnt.Font, buffer *sfnt.Buffer, size fract.Unit) fract.Unit {
	self.fonts = append(self.fonts, font)
	return self.DefaultSizer.LineHeight(font, buffer, size)
}

func (self *lineMetricRecorder) LineAdvance(font *sfnt.Font, buffer *sfnt.Buffer, size fract.Unit, nth int) fract.Unit {
	self.fonts = append(self.fonts, font)
	return self.DefaultSizer.LineAdvance(font, buffer, size, nth)
}

// recordedOnly reports whether every recorded font is the given one. It's
// false when nothing was recorded, so a check can't pass without line metrics.
func (self *lineMetricRecorder) recordedOnly(font *sfnt.Font) bool {
	if len(self.fonts) == 0 {
		return false
	}
	for _, recordedFont := range self.fonts {
		if recordedFont != font {
			return false
		}
	}
	return true
}

// TestMeasureScriptFontsLineMetrics verifies that line metrics come from the
// primary font, even when a line ends with a glyph from a script font, with
// and without wrapping. The text must be as tall as the same number of lines
// in the primary font.
func TestMeasureScriptFontsLineMetrics(t *testing.T) {
	primary, han := scriptFontsForTest(t)
	renderer := NewRenderer()
	renderer.SetFont(primary)
	twoLines := renderer.Measure("a\na").Height()
	threeLines := renderer.Measure("a\na\na").Height()
	renderer.Script().SetFont(unicode.Han, han)
	metrics := &lineMetricRecorder{}
	renderer.SetSizer(metrics)
	for _, direction := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(direction)
		if height := renderer.Measure(lineMetricsText).Height(); height != twoLines {
			t.Fatalf("%v: expected height %v, got %v", direction, twoLines, height)
		}
		widthLimit := lineMetricsWrapLimit(t, renderer)
		if height := renderer.MeasureWithWrap(lineMetricsText, widthLimit).Height(); height != threeLines {
			t.Fatalf("%v (wrapped): expected height %v, got %v", direction, threeLines, height)
		}
	}
	if !metrics.recordedOnly(primary) {
		t.Fatalf("line metrics used a font other than the primary: %v", metrics.fonts)
	}
}

// kernRecorder is a sizer that records each Kern() call.
type kernRecorder struct {
	sizer.DefaultSizer
	pairs [][2]sfnt.GlyphIndex
}

func (self *kernRecorder) Kern(font *sfnt.Font, buffer *sfnt.Buffer, size fract.Unit, g1, g2 sfnt.GlyphIndex) fract.Unit {
	self.pairs = append(self.pairs, [2]sfnt.GlyphIndex{g1, g2})
	return 0
}

// TestMeasureScriptFontsKerning verifies that glyphs from different fonts don't
// kern with each other. After a font change, the renderer kerns the new glyph
// against glyph 0, which has no kerning pairs. In RTL the new glyph is placed
// to the left of the previous one, so glyph 0 is the second glyph of the pair.
func TestMeasureScriptFontsKerning(t *testing.T) {
	primary, han := scriptFontsForTest(t)
	kerns := &kernRecorder{}
	renderer := NewRenderer()
	renderer.SetFont(primary)
	renderer.Script().SetFont(unicode.Han, han)
	renderer.SetSizer(kerns)
	for _, direction := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(direction)
		kerns.pairs = kerns.pairs[:0]
		renderer.Measure("a中")
		previous := 0 // index of the previous glyph within the kerning pair
		if direction == RightToLeft {
			previous = 1
		}
		if len(kerns.pairs) != 1 || kerns.pairs[0][previous] != 0 {
			t.Fatalf("%v: expected a single kerning pair with glyph 0 as the previous glyph, got %v", direction, kerns.pairs)
		}
	}
}

// TestMeasureScriptFontsAllocations verifies that measuring text that switches
// between script fonts doesn't allocate.
func TestMeasureScriptFontsAllocations(t *testing.T) {
	primary, han := scriptFontsForTest(t)

	renderer := NewRenderer()
	renderer.SetFont(primary)
	renderer.Script().SetFont(unicode.Han, han)
	text := scriptCaseText(scriptCases[0])
	forceWrapLimit := renderer.Measure(text).IntWidth() / 2 // wraps the first line

	allocs := testing.AllocsPerRun(100, func() {
		renderer.Measure(text)
		renderer.MeasureWithWrap(text, forceWrapLimit)
	})
	if allocs != 0 {
		t.Fatalf("expected no allocations, got %v", allocs)
	}
}

// BenchmarkScriptFontsMeasure compares Measure per rune with and without
// script fonts.
func BenchmarkScriptFontsMeasure(b *testing.B) {
	primary, han := scriptFontsForTest(b)

	// Keep the cases stable so itemization optimizations can be compared over time.
	cases := []struct {
		name string
		line string
	}{
		{"latin-ASCII", "The quick brown fox jumps over the lazy dog, again and again."},
		{"latin-ext", "Calle de Oslo, 53 · 28922 Alcorcón — café, niño, über alles."},
		{"mixed", "Jack 中文 (是) 你 is one, 一 and 中文你是一. The end 中."},
		{"punctuation", "a .#!/._-()[]{}<>~^*+=|;:,? b .#!/._-()[]{}<>~^*+=|;:,? c"},
		{"common-lines", "1234567890 .,;:!? -+*/() 1234567890 .,;:!? -+*/() 1234567"},
		{"long-common-run", "a" + strings.Repeat(".#!/._-()", 30) + "b"},
	}
	for _, c := range cases {
		text := c.line
		for i := 0; i < 7; i++ {
			text += "\n" + c.line
		}
		runes := len([]rune(text))
		for _, scripts := range []bool{false, true} {
			// create a new renderer each time, no cache needed, we are only measuring
			renderer := NewRenderer()
			renderer.SetSize(16)
			renderer.SetFont(primary)
			renderer.Glyph().SetMissHandler(OnMissNotdef) // the test fonts may lack some glyphs, which doesn't matter for timing
			name := c.name + "/no-scripts"
			if scripts {
				name = c.name + "/scripts"
				renderer.Script().SetFont(unicode.Han, han)
			}
			b.Run(name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = renderer.Measure(text)
				}
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*runes), "ns/rune")
			})
		}
	}
}
