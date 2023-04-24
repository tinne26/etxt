package etxt

import "strconv"
//import "image/color"

import "golang.org/x/image/font/sfnt"

// Helper types, wrappers, aliases and functions.

// Renderers can have their text direction configured as
// left-to-right or right-to-left. See [RendererComplex.SetDirection]().
//
// Directions can be casted directly to [unicode/bidi] directions:
//   bidi.Direction(etxt.LeftToRight).
//
// [unicode/bidi]: https://pkg.go.dev/golang.org/x/text/unicode/bidi
type Direction int8
const (
	LeftToRight Direction = iota
	RightToLeft
)

// A handy type alias for sfnt.Font so you don't need to
// import it when already working with etxt.
type Font = sfnt.Font

// Glyph indices are used to specify which font glyph are we working
// with. Glyph indices are a low level construct that most users of
// etxt dont't have to deal with, but they are important as they can
// be used to reference font glyphs that don't have any direct mapping
// to unicode code points.
//
// Support for glyph indices (and not only runes), therefore, is important
// in order to make renderers usable with [text shapers] and complex scripts.
// TODO: But I'm not really offering much support for that yet...
// TODO: probably kill glyph index alias? Why the hell do I need it?
//       it comes down to advanced use-cases that I probably don't want
//       to actively cater to. I mean, ok, I could use this for cache
//       implementations and stuff, but then I'm already importing sfnt.
//       What about using uint16 directly? So sfnt and etxt have compatible
//       representations, even if they don't... no, that sounds like bad
//       practice, straightforward. I'll only use those glyphs with sfnt
//       fonts for the moment anyway, so...
//
// [text shapers]: https://github.com/tinne26/etxt/blob/main/docs/shaping.md
//type GlyphIndex = sfnt.GlyphIndex

// --- misc helpers ---

// Utility method to create opaque RGBA colors.
// func RGB(r, g, b uint8) color.RGBA {
// 	return color.RGBA{r, g, b, 255}
// }

func runeToUnicodeCode(r rune) string {
	return "\\u" + strconv.FormatInt(int64(r), 16)
}

func ensureSliceSize[T any](slice []T, size int) []T {
	// easy case: slice is big enough already
	if len(slice) >= size { return slice }

	// see if we have enough capacity...
	if cap(slice) >= size {
		return slice[ : size]
	} else { // ...or allocate new slice otherwise
		newSlice := make([]T, size)
		if len(slice) > 0 { // preserve previous data
			copy(newSlice, slice)
		}
		return newSlice
	}
}
