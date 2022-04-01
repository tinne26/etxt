//go:build !gtxt

package ecache

const constMaskSizeFactor = 192

// Returns an approximation of the GlyphMask's size in bytes.
//
// With Ebiten, the exact amount of mipmaps and helper fields is
// not known, so the values may not be very representative of actual
// memory usage. With gtxt, the returned values are precise.
func GlyphMaskByteSize(mask GlyphMask) uint32 {
	if mask == nil { return constMaskSizeFactor }
	w, h := mask.Image.Size()
	return maskDimsByteSize(w, h)
}

func maskDimsByteSize(width, height int) uint32 {
	return uint32(width*height)*4 + constMaskSizeFactor
}
