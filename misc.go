package etxt

import "strconv"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// Helper types, wrappers, aliases and functions.

// A handy type alias for [sfnt.Font] so you don't need to
// import it when already working with etxt.
type Font = sfnt.Font

// See [RendererTwine.RegisterFont]() and related functions.
//
// When using multiple fonts, you are encouraged to define
// and use your own named constants in the relevant context,
// like:
//   const (
//       RegularFont FontIndex = iota
//       BoldFont
//       ItalicFont
//   )
type FontIndex uint8
const (
	NextFontIndex FontIndex = 255 // see RendererTwine.RegisterFont()
)

// Quantization levels are used to control the trade-off between
// memory usage and glyph positioning precision. Less theoretically:
//  - A font is a collection of glyph outlines. Each time we want to
//    draw a glyph at a specific size, we need to rasterize its outline
//    into a bitmap. That's expensive, so we want to cache the bitmaps.
//  - Now we have some text and we want to start drawing each letter
//    one after another, but... most of the time, the glyph positions
//    won't align to the pixel grid perfectly. Should we force this
//    alignment? Or should we rasterize glyph variants for each subposition?
// The answer is that it depends on what we are trying to do. Quantization
// levels allow us to adjust the number of positional variants we want to
// allow for each glyph. More variants means more memory usage. Less
// variants mean less positioning precision.
//
// Quantization levels can be configured with [RendererFract.SetHorzQuantization]()
// and [RendererFract.SetVertQuantization](). Stick to the defaults unless you
// are trying to animate text with movement or know what you are doing.
//
// Only the equispaced quantization values are given. Other values like
// [fract.Unit](22) (which approximates one third of a pixel, ceil(64/3))
// could also work in theory, but in practice they lead to all kinds of
// complications that are simply not worth it.
const (
	QtNone = fract.Unit( 1) // full glyph position resolution (1/64ths of a pixel)
	Qt32th = fract.Unit( 2) // quantize glyph positions to 1/32ths of a pixel
	Qt16th = fract.Unit( 4) // quantize glyph positions to 1/16ths of a pixel
	Qt8th  = fract.Unit( 8) // quantize glyph positions to 1/ 8ths of a pixel
	Qt4th  = fract.Unit(16) // quantize glyph positions to 1/ 4ths of a pixel
	QtHalf = fract.Unit(32) // quantize glyph positions to half of a pixel
	QtFull = fract.Unit(64) // full glyph position quantization
)

// Renderers can have their text direction configured as
// left-to-right or right-to-left. See [Renderer.SetDirection]().
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

// Returns the string representation of the [Direction]
// constant (e.g., "LeftToRight", "RightToLeft").
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

// replace with max() when minimum version reaches go1.21
func maxInt(a, b int) int {
	if a >= b { return a }
	return b
}

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
