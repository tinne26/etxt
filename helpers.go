package etxt

import "strconv"
import "image"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/ecache"
import "github.com/tinne26/etxt/internal"

// TODO: create GlyphMask read-only type with .Image() and Bounds()
//       and store that on cache directly instead of GlyphMask?
//       The bounds are not heavy to store compared to the image. The
//       overhead is not severe, and while the bounds are rarely practical,
//       they add quite a bit in terms of the completeness for etxt. Also,
//       since I already need the extra struct for ebiten...

// This file contains many helper types, wrappers, aliases and
// other minor elements required to make this whole package work.

// An alias for sfnt.Font so you don't need to import sfnt yourself
// when working with etxt.
type Font = sfnt.Font

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
type GlyphIndex = sfnt.GlyphIndex

// A GlyphMask is the image that results from rasterizing a glyph.
// You rarely need to use GlyphMasks directly unless using advanced
// functions.
//
// Without Ebitengine (gtxt version), GlyphMask defaults to [*image.Alpha].
// The image bounds are adjusted to allow drawing the glyph at its
// intended position. In particular, bounds.Min.Y is typically
// negative, with y = 0 corresponding to the glyph's baseline, y < 0
// to the ascending portions and y > 0 to the descending ones.
//
// With Ebitengine, GlyphMask defaults to a struct with the following fields:
//   Image *ebiten.Image // the actual glyph image
//   XOffset int         // horizontal drawing offset
//   YOffset int         // vertical drawing offset
type GlyphMask = internal.GlyphMask

// Quantization modes can be used to tell a [Renderer] whether it should
// operate aligning glyphs to the pixel grid or not. When not following
// the pixel grid and operating at a fractional pixel level, glyphs can be
// drawn in up to 64 positions per axis.
//
// Read the [quantization document] if you need more details.
//
// [quantization document]: https://github.com/tinne26/etxt/blob/main/docs/quantization.md
type QuantizationMode uint8
const (
	QuantizeNone QuantizationMode = 0
	QuantizeVert QuantizationMode = 1
	QuantizeFull QuantizationMode = 2
)

// Text alignment types.

type VertAlign int8
type HorzAlign int8

// Vertical align constants for renderer operations. See
// [Renderer.SetAlign] for additional details.
const (
	Top      VertAlign = 0
	YCenter  VertAlign = 1
	Baseline VertAlign = 2
	Bottom   VertAlign = 3
)

// Horizontal align constants for renderer operations. See
// [Renderer.SetAlign] for additional details.
const (
	Left    HorzAlign = 0
	XCenter HorzAlign = 1
	Right   HorzAlign = 2
)

// Renderers can have their text direction configured as
// left-to-right or right-to-left.
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

// Creates a new cache for font glyphs. For more details on how to use
// this new cache with renderers, see [Renderer.SetCacheHandler]() .
//
// This function will panic if maxBytes < 1024 or crypto/rand fails. If
// you need to handle those errors, see [ecache.NewDefaultCache]() instead.
func NewDefaultCache(maxBytes int) *ecache.DefaultCache {
	cache, err := ecache.NewDefaultCache(maxBytes)
	if err != nil { panic(err) }
	return cache
}

// RectSize objects are used to store the results of text sizing operations.
// If you need to use the fixed.Int26_6 values directly and would like more
// context on them, read [this document]. Otherwise, you can obtain RectSize
// dimensions as int values like this:
//    rect   := txtRenderer.SelectionRect(text)
//    width  := rect.Width.Ceil()
//    height := rect.Height.Ceil()
//
// [this document]: https://github.com/tinne26/etxt/blob/main/docs/fixed-26-6.md
type RectSize struct { Width fixed.Int26_6 ; Height fixed.Int26_6 }

// Deprecated: prefer RectSize.Width.Ceil() directly instead.
//
// Get the RectSize's width in whole pixels.
func (self RectSize) WidthCeil()  int { return self.Width.Ceil()  }

// Deprecated: prefer RectSize.Width.Ceil() directly instead.
//
// Get the RectSize's height in whole pixels.
func (self RectSize) HeightCeil() int { return self.Height.Ceil() }

// Returns the RectSize as an image.Rectangle with origin at (0, 0).
// 
// Ebitengine and other projects often expect image.Rectangle objects
// as arguments in their API calls, so this method is offered as a
// handy conversion shortcut.
func (self RectSize) ImageRect() image.Rectangle {
	return image.Rect(0, 0, self.Width.Ceil(), self.Height.Ceil())
}

// --- misc helpers ---

func runeToUnicodeCode(r rune) string {
	return "\\u" + strconv.FormatInt(int64(r), 16)
}
