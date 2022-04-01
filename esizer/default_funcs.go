package esizer

import "strconv"

import "golang.org/x/image/font"
import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

// A default method to respond to font metrics requests. Can be used
// while implementing your own sizers.
func DefaultMetricsFunc(f *Font, size fixed.Int26_6, buffer *sfnt.Buffer) font.Metrics {
	metrics, err := f.Metrics(buffer, size, font.HintingNone)
	if err != nil { panic("font.Metrics error: " + err.Error()) }
	return metrics
}

// A default method to respond to glyph advance requests. Can be used
// while implementing your own sizers.
func DefaultAdvanceFunc(f *Font, glyphIndex GlyphIndex, size fixed.Int26_6, buffer *sfnt.Buffer) fixed.Int26_6 {
	advance, err := f.GlyphAdvance(buffer, glyphIndex, size, font.HintingNone)
	if err == nil { return advance }

	msg := "font.GlyphAdvance(index = " + strconv.Itoa(int(glyphIndex))
	msg += ") error: " + err.Error()
	panic(msg)
}

// A default method to respond to glyph kern requests. Can be used
// while implementing your own sizers.
func DefaultKernFunc(f *Font, prevGlyphIndex GlyphIndex, currGlyphIndex GlyphIndex, size fixed.Int26_6, buffer *sfnt.Buffer) fixed.Int26_6 {
	kern, err := f.Kern(buffer, prevGlyphIndex, currGlyphIndex, size, font.HintingNone)
	if err == nil { return kern }
	if err == sfnt.ErrNotFound { return 0 }

	msg := "font.Kern failed for glyphs with indices "
	msg += strconv.Itoa(int(prevGlyphIndex)) + " and "
	msg += strconv.Itoa(int(currGlyphIndex)) + ": " + err.Error()
	panic(msg)
}
