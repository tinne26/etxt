//go:build !gtxt

package cache

import "github.com/hajimehoshi/ebiten/v2"

// Same as [etxt.GlyphMask], redefined locally for improved clarity
// and consistency with the etxt parent package when defining caches
// and the [GlyphCacheHandler] interface.
//
// [etxt.GlyphMask]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.6#GlyphMask
type GlyphMask = *ebiten.Image

// Based on Ebitengine internals.
const constMaskSizeFactor = 192

// Returns an approximation of a [GlyphMask] size in bytes.
//
// With Ebitengine, the exact amount of mipmaps and helper fields is
// not known, so the values may not be completely accurate, and should
// be treated as a lower bound. With gtxt, the returned values are
// exact.
func GlyphMaskByteSize(mask GlyphMask) uint32 {
	if mask == nil { return constMaskSizeFactor }
	w, h := mask.Size()
	return maskDimsByteSize(w, h)
}

func maskDimsByteSize(width, height int) uint32 {
	return uint32(width*height)*4 + constMaskSizeFactor
}

// used for testing purposes
func newEmptyGlyphMask(width, height int) GlyphMask {
	return GlyphMask(ebiten.NewImage(width, height))
}
