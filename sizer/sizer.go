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
	//         some custom rasterizers. Uncommon but possible.
	//         I also think they can be generally helpful.

	// Returns the ascent of the given font, at the given size,
	// as an absolute value.
	//
	// The given font and sizes must be consistent with the
	// latest Sizer.NotifyChange() call.
	Ascent(*Font, *Buffer, fract.Unit) fract.Unit

	// Returns the descent of the given font, at the given size,
	// as an absolute value.
	//
	// The given font and sizes must be consistent with the
	// latest Sizer.NotifyChange() call.
	Descent(*Font, *Buffer, fract.Unit) fract.Unit

	// Returns the line gap of the given font, at the given size,
	// as an absolute value.
	//
	// The given font and sizes must be consistent with the
	// latest Sizer.NotifyChange() call.
	LineGap(*Font, *Buffer, fract.Unit) fract.Unit

	// Utility method equivalent to Ascent() + Descent() + LineGap().
	LineHeight(*Font, *Buffer, fract.Unit) fract.Unit
	
	// Returns the line advance of the given font at the given size.
	//
	// The given font and the size must be consistent with the
	// latest Sizer.NotifyChange() call.
	//
	// The given int indicates that this is the nth consecutive
	// call to the method (consecutive line breaks). In most cases,
	// the value will be 1. Values below 1 are invalid. Values
	// can only be strictly increasing by +1.
	LineAdvance(*Font, *Buffer, fract.Unit, int) fract.Unit

	// Returns the advance of the given glyph for the given font
	// and size.
	//
	// The given font and the size must be consistent with the
	// latest Sizer.NotifyChange() call.
	GlyphAdvance(*Font, *Buffer, fract.Unit, GlyphIndex) fract.Unit

	// Returns the kerning value between two glyphs of the given font
	// and size.
	//
	// The given font and the size must be consistent with the
	// latest Sizer.NotifyChange() call.
	Kern(*Font, *Buffer, fract.Unit, GlyphIndex, GlyphIndex) fract.Unit

	// Must be called to sync the state of the sizer and allow it
	// to do any caching it may want to do in relation to the given
	// active font or size.
	NotifyChange(*Font, *Buffer, fract.Unit)
}
