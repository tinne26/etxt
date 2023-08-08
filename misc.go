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
	textDirectionUnexportedMixed
	textDirectionUnexportedNeutral
)

func (self Direction) String() string {
	switch self {
	case LeftToRight: return "LeftToRight"
	case RightToLeft: return "RightToLeft"
	case textDirectionUnexportedMixed: return "Mixed"
	case textDirectionUnexportedNeutral: return "Neutral"
	default:
		return "UnknownTextDirection"
	}
}

// A handy type alias for sfnt.Font so you don't need to
// import it when already working with etxt.
type Font = sfnt.Font

// See [RendererComplex.RegisterFont]() and related functions.
//
// When using multiple fonts, you are encouraged to define
// and use your own named constants in the relevant context,
// like:
//   const (
//	      RegularFont FontIndex = iota
//       BoldFont
//       ItalicFont
//   )
type FontIndex uint8

// Glyph indices are used to specify which font glyph are we working
// with. Glyph indices are a low level construct that most users of
// etxt dont't have to deal with, but they are important as they can
// be used to reference font glyphs that don't have any direct mapping
// to unicode code points.
//
// Support for glyph indices (and not only runes), therefore, is important
// in order to make renderers usable with [text shapers] and complex scripts.
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
