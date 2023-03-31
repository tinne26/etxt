package sizer

import . "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/fract"

// When drawing or traversing glyphs, we need some information
// related to the "font metrics". For example, how much we need
// to advance after drawing a glyph or what's the kerning between
// a specific pair of glyphs.
//
// Sizers are the interface that renderers use to obtain that
// information.
//
// You rarely need to care about sizers, but they can be useful
// in the following cases:
//  - Customize line height or advances.
//  - Disable kerning or adjust horizontal spacing.
//  - Make full size adjustments for a custom rasterizer (e.g.,
//    a rasterizer that puts glyphs into boxes, bubbles or frames).
type Sizer interface {
	// Notice: while Ascent(), Descent() and LineGap() may
	//         seem superfluous, they can be necessary for
	//         some custom rasterizers. Rare but possible.

	// Returns the ascent of the given font, at the given size,
	// as an absolute value.
	//
	// The given font and sizes must be consistent with the
	// latest NotifyFontChange() and NotifySizeChange() calls.
	Ascent(*Font, *Buffer, fract.Unit) fract.Unit

	// Returns the descent of the given font, at the given size,
	// as an absolute value.
	//
	// The given font and sizes must be consistent with the
	// latest NotifyFontChange() and NotifySizeChange() calls.
	Descent(*Font, *Buffer, fract.Unit) fract.Unit

	// Returns the line gap of the given font, at the given size,
	// as an absolute value.
	//
	// The given font and sizes must be consistent with the
	// latest NotifyFontChange() and NotifySizeChange() calls.
	LineGap(*Font, *Buffer, fract.Unit) fract.Unit

	// Utility method equivalent to Ascent() + Descent() + LineGap().
	LineHeight(*Font, *Buffer, fract.Unit) fract.Unit
	
	// Returns the line advance of the given font at the given size.
	//
	// The given font and the size must be consistent with the
	// latest NotifyFontChange() and NotifySizeChange() calls.
	//
	// The returned value may vary for consecutive line breaks or to
	// take into account different text sizes within a single line.
	// For this reason, once a draw or measuring operation is done,
	// ResetLineState() must be called to reset any possible lingering
	// state and avoid collisions with later operations.
	LineAdvance(*Font, *Buffer, fract.Unit) fract.Unit

	// See GetLineAdvance() documentation.
	ResetLineState()

	// Returns the advance of the given glyph for the given font
	// and size.
	//
	// The given font and the size must be consistent with the
	// latest NotifyFontChange() and NotifySizeChange() calls.
	GlyphAdvance(*Font, *Buffer, fract.Unit, GlyphIndex) fract.Unit

	// Returns the kerning value between two glyphs of the given font
	// and size.
	//
	// The given font and the size must be consistent with the
	// latest NotifyFontChange() and NotifySizeChange() calls.
	Kern(*Font, *Buffer, fract.Unit, GlyphIndex, GlyphIndex) fract.Unit

	// Must be called to sync the state of the sizer and allow it
	// to do any caching it may want to do in relation to the given
	// active font or size.
	NotifyChange(*Font, *Buffer, fract.Unit)
}
