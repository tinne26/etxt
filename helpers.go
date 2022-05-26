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

// Support for glyph indices (and not only runes) is important in order
// to make renderers usable with [text shapers] and complex scripts.
//
// [text shapers]: https://github.com/tinne26/etxt/blob/main/docs/shaping.md
type GlyphIndex = sfnt.GlyphIndex

// A GlyphMask is the image that results from rasterizing a glyph.
// You rarely need to use GlyphMasks directly unless using advanced
// functions.
//
// Without Ebitengine (gtxt version), GlyphMask defaults to *image.Alpha.
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

// Quantization modes can be used to tell a Renderer whether it should
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
// Renderer.SetAlign for additional details.
const (
	Top      VertAlign = 0
	YCenter  VertAlign = 1
	Baseline VertAlign = 2
	Bottom   VertAlign = 3
)

// Horizontal align constants for renderer operations. See
// Renderer.SetAlign for additional details.
const (
	Left    HorzAlign = 0
	XCenter HorzAlign = 1
	Right   HorzAlign = 2
)

// Renderers can have their text direction configured as
// left-to-right or right-to-left.
//
// Directions can be casted directly to [unicode/bidi] directions, e.g:
//   bidi.Direction(etxt.LeftToRight).
//
// [unicode/bidi]: https://pkg.go.dev/golang.org/x/text/unicode/bidi
type Direction int8
const (
	LeftToRight Direction = iota
	RightToLeft
)

// Creates a new cache for glyphs. You can call NewHandler() on the
// returned cache to obtain a cache handler to pass to your Renderer.
// A handler can only be used with a single Renderer, but you can create
// multiple handlers for the same underlying cache.
//
// Will panic if maxBytes < 1024 or crypto/rand fails. If you want
// to handle those errors or learn more, see the ecache subpackage.
func NewDefaultCache(maxBytes int) *ecache.DefaultCache {
	cache, err := ecache.NewDefaultCache(maxBytes)
	if err != nil { panic(err) }
	return cache
}

// RectSize objects are used to store the results of text sizing operations.
type RectSize struct { Width fixed.Int26_6 ; Height fixed.Int26_6 }

// Get the RectSize's width in whole pixels.
func (self RectSize) WidthCeil()  int { return self.Width.Ceil()  }

// Get the RectSize's height in whole pixels.
func (self RectSize) HeightCeil() int { return self.Height.Ceil() }

// Handy conversion method.
func (self RectSize) ImageRect() image.Rectangle {
	return image.Rect(0, 0, self.WidthCeil(), self.HeightCeil())
}
// func (self RectSize) RoundedWidth() int {
// 	return efixed.ToIntHalfUp(self.Width)
// }
// func (self RectSize) RoundedHeight() int {
// 	return efixed.ToIntHalfUp(self.Height)
// }

// --- misc helpers ---

func runeToUnicodeCode(r rune) string {
	return "\\u" + strconv.FormatInt(int64(r), 16)
}
