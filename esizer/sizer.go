package esizer

import "golang.org/x/image/font"
import "golang.org/x/image/math/fixed"

// When drawing or traversing glyphs, we need some information
// related to the "font metrics". For example, how much we need
// to advance after drawing a glyph or what's the kerning between
// a specific pair of glyphs.
//
// Sizers are the interface that Renderers use to access that
// information.
//
// You rarely need to care about Sizers, but they can be useful
// in the following cases:
//  - Disable kerning and adjust horizontal spacing.
//  - Make full size adjustments for a custom rasterizer (e.g.,
//    a rasterizer that puts glyphs into boxes, bubbles or frames).
type Sizer interface {
	Metrics(*Font, fixed.Int26_6) font.Metrics
	Advance(*Font, GlyphIndex, fixed.Int26_6) fixed.Int26_6
	Kern(*Font, GlyphIndex, GlyphIndex, fixed.Int26_6) fixed.Int26_6

	// We could have a method for bounds too, but we
	// don't need it in etxt, so it doesn't exist yet.
	//Bounds(*Font, GlyphIndex, fixed.Int26_6, *sfnt.Buffer) fixed.Rect26_6
}
