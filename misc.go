package etxt

import (
	"strconv"

	"github.com/tinne26/etxt/fract"
	"golang.org/x/image/font/sfnt"
)

// Helper types, wrappers, aliases and functions.

// A handy type alias for [sfnt.Font] so you don't need to
// import it separately when working with etxt.
type Font = sfnt.Font

// Glyph indices are used to specify which font glyph are we working
// with. They allow us to reference font glyphs that aren't mapped
// to any unicode code point (rune).
//
// This type is a direct alias of [sfnt.GlyphIndex] so you don't have
// to import a separate package when working with [TwineMotionFunc] or
// custom functions for [RendererGlyph.SetDrawFunc](). Everywhere else
// in the documentation [sfnt.GlyphIndex] is used, but they are completely
// interchangeable.
type GlyphIndex = sfnt.GlyphIndex

// See [RendererTwine.RegisterFont]() and related functions.
//
// When using multiple fonts, you are encouraged to define
// and use your own named constants within the relevant context.
// For example:
//
//	const (
//	    RegularFont etxt.FontIndex = iota
//	    BoldFont
//	    ItalicFont
//	)
type FontIndex uint8

// Special value for [RendererTwine.RegisterFont]().
const NextFontIndex FontIndex = 255

// Quantization levels are used to control the trade-off between
// memory usage and glyph positioning precision. Less theoretically:
//   - A font is a collection of glyph outlines. Each time we want to
//     draw a glyph at a specific size, we need to rasterize its outline
//     into a bitmap. That's expensive, so we want to cache the bitmaps.
//   - Now we have some text and we want to start drawing each letter
//     one after another, but... most of the time, the glyph positions
//     won't align to the pixel grid perfectly. Should we force this
//     alignment? Or should we rasterize glyph variants for each subposition?
//
// The answer is that it depends on what we are trying to do. Quantization
// levels allow us to adjust the number of positional variants we want to
// consider for each glyph. More variants mean higher memory usage. Less
// variants mean lower positioning precision.
//
// Quantization levels can be adjusted with [RendererFract.SetHorzQuantization]()
// and [RendererFract.SetVertQuantization](), though in general they are not
// something you should be touching unless you know what you are doing.
//
// Only the equispaced quantization values are given. Other values like
// [fract.Unit](22) (which approximates one third of a pixel, ceil(64/3))
// could also work in theory, but in practice they lead to implementation
// complications that are simply not worth it.
const (
	QtNone = fract.Unit(1)  // full glyph position resolution (1/64ths of a pixel)
	Qt32th = fract.Unit(2)  // quantize glyph positions to 1/32ths of a pixel
	Qt16th = fract.Unit(4)  // quantize glyph positions to 1/16ths of a pixel
	Qt8th  = fract.Unit(8)  // quantize glyph positions to 1/ 8ths of a pixel
	Qt4th  = fract.Unit(16) // quantize glyph positions to 1/ 4ths of a pixel
	QtHalf = fract.Unit(32) // quantize glyph positions to half of a pixel
	QtFull = fract.Unit(64) // full glyph position quantization
)

// Renderers can have their text direction configured as
// left-to-right or right-to-left. See [Renderer.SetDirection]()
// for further context and details.
//
// If necessary, directions can also be casted directly to
// [unicode/bidi] directions:
//
//	bidi.Direction(etxt.LeftToRight).
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
// (e.g., "LeftToRight", "RightToLeft").
func (self Direction) String() string {
	switch self {
	case LeftToRight:
		return "LeftToRight"
	case RightToLeft:
		return "RightToLeft"
	case textDirectionUnexportedMixed:
		return "Mixed"
	case textDirectionUnexportedNeutral:
		return "Neutral"
	default:
		return "UnknownTextDirection"
	}
}

// --- misc helpers ---

// can replace with max() when minimum version reaches go1.21
func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

func runeToUnicodeCode(r rune) string {
	return "\\u" + strconv.FormatInt(int64(r), 16)
}

func ensureSliceSize[T any](slice []T, size int) []T {
	// easy case: slice is big enough already
	if len(slice) >= size {
		return slice
	}

	// see if we have enough capacity...
	if cap(slice) >= size {
		return slice[:size]
	} else { // ...or grow slice otherwise
		slice = slice[:cap(slice)]
		growth := size - len(slice)
		return append(slice, make([]T, growth)...)
	}
}
