//go:build gtxt

package cache

import "image"

// Alias for etxt.GlyphMask.
type GlyphMask = *image.Alpha

const constMaskSizeFactor = 56

func GlyphMaskByteSize(mask GlyphMask) uint32 {
	if mask == nil {
		return constMaskSizeFactor
	}
	w, h := mask.Rect.Dx(), mask.Rect.Dy()
	return maskDimsByteSize(w, h)
}

func maskDimsByteSize(width, height int) uint32 {
	return uint32(width*height) + constMaskSizeFactor
}

// used for testing purposes
func newEmptyGlyphMask(width, height int) GlyphMask {
	return GlyphMask(image.NewAlpha(image.Rect(0, 0, width, height)))
}
