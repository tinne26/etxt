package mask

import "image"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

var _ Rasterizer = (*SharpRasterizer)(nil)

// A rasterizer that quantizes all glyph mask values to fully opaque
// or fully transparent. Its primary use-case is to make scaled pixel
// art fonts look sharper through the elimination of blurry edges.
//
// Since the implementation leverages type embedding, the available methods
// are the same as the ones for [DefaultRasterizer], even if they do not
// appear explicitly in the documentation.
type SharpRasterizer struct { DefaultRasterizer }

// Satisfies the [Rasterizer] interface.
func (self *SharpRasterizer) Rasterize(outline sfnt.Segments, origin fract.Point) (*image.Alpha, error) {
	mask, err := self.DefaultRasterizer.Rasterize(outline, origin)
	if err != nil { return mask, err }
	for i := 0; i < len(mask.Pix); i++ {
		// we use 128 as the threshold, but if you want another value,
		// just copy paste the extremely short code and set your own
		// or make it customizable
		if mask.Pix[i] < 128 {
			mask.Pix[i] = 0
		} else {
			mask.Pix[i] = 255
		}
	}
	return mask, err
}
