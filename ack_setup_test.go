package etxt

// This file gives tests access to the fonts placed in font/test/, picked by
// the glyphs each test needs, and provides some helper methods.

import (
	"embed"
	"os"
	"testing"

	"github.com/tinne26/etxt/internal/testfont"
	"golang.org/x/image/font/sfnt"
)

//go:embed font/test/*
var testfs embed.FS

var testFonts = testfont.NewDir(testfs, "font/test")

func TestMain(m *testing.M) {
	os.Exit(testFonts.ReportSkips(m.Run()))
}

// samples for picking test fonts by glyph coverage
const testLatinSample = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .,:;!?()[]'\""
const testHanSample = "中文你是一"

// testFontsWithRunes returns a different test font for each of the given
// samples, with glyphs for all the runes in that sample. An empty sample
// accepts any font. See [testfont.Dir.WithRunes] for details.
func testFontsWithRunes(tb testing.TB, samples ...string) []*sfnt.Font {
	tb.Helper()
	picked := testFonts.WithRunes(tb, samples...)
	fonts := make([]*sfnt.Font, len(picked))
	for i := range picked {
		fonts[i] = picked[i].Font
	}
	return fonts
}

func doesNotPanic(function func()) (didNotPanic bool) {
	didNotPanic = true
	defer func() { didNotPanic = (recover() == nil) }()
	function()
	return
}
