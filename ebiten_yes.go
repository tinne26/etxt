//go:build !gtxt

package etxt

import "image"
import "image/color"

//import "golang.org/x/image/font/sfnt"
import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt/fract"

// Alias to allow compiling the package without Ebitengine (-tags gtxt).
//
// Without Ebitengine, Target defaults to [image/draw.Image].
type Target = *ebiten.Image

// A GlyphMask is the image that results from rasterizing a glyph.
// You rarely need to use glyph masks directly unless you are working
// with advanced functions or custom caches.
//
// Without Ebitengine, GlyphMask defaults to [*image.Alpha]. The
// image bounds are adjusted to allow drawing the glyph at its
// intended position. In particular, bounds.Min.Y is typically
// negative, with y = 0 corresponding to the glyph's baseline, y < 0
// to the ascending portions and y > 0 to the descending ones.
//
// With Ebitengine, GlyphMask defaults to [*ebiten.Image].
type GlyphMask = *ebiten.Image

// The blend mode specifies how to compose colors when drawing glyphs:
//  - Without Ebitengine, the blend mode can be BlendOver, BlendReplace,
//    BlendAdd, BlendSub, BlendMultiply, BlendCut and BlendHue.
//  - With Ebitengine, the blend mode is Ebitengine's [Blend].
//
// I only ever change blend modes to make cutout text, but there's a
// lot of weird people out there, what can I say.
//
// [Blend]: https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Blend
type BlendMode = ebiten.Blend

// Underlying default glyph drawing function for renderers.
// Can be overridden with Renderer.Glyph().SetDrawFunc(...).
func (self *Renderer) defaultDrawFunc(target Target, origin fract.Point, mask GlyphMask) {
	if mask == nil { return } // spaces and empty glyphs will be nil

	// TODO: maybe switch to DrawTriangles, but specially, move opts out (tricky due to gtxt)
	//       and have color set only when necessary, translations reset, blend mode set only
	//       when necessary, etc. Or maybe not. At least write a quick benchmark to see the
	//       impact of moving opts out.
	opts := ebiten.DrawImageOptions{}
	srcRect := mask.Bounds()
	opts.GeoM.Translate(float64(origin.X.ToIntFloor() + srcRect.Min.X), float64(origin.Y.ToIntFloor() + srcRect.Min.Y))
	r, g, b, a := colorToFloat32(self.state.fontColor)
	opts.ColorScale.Scale(r, g, b, a)
	opts.Blend = self.state.blendMode
	target.DrawImage(mask, &opts)
}

// Convert a color to its float64 [0, 1.0] components.
// This could actually be memorized to make DefaultDrawFunc work better
// in most cases, but I don't know if it's worth the extra complexity.
//
// Note: I benchmarked this and it's typically visibly faster than the
// second direct branch alone.
func colorToFloat32(subject color.Color) (float32, float32, float32, float32) {
	rgbaColor, isRGBA := subject.(color.RGBA)
	if isRGBA {
		r, g, b, a := rgbaColor.R, rgbaColor.G, rgbaColor.B, rgbaColor.A
		return float32(r)/255, float32(g)/255, float32(b)/255, float32(a)/255
	} else {
		r, g, b, a := subject.RGBA()
		return float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535
	}
}

// helper function required when working with ebitengine images
func convertAlphaImageToGlyphMask(alpha *image.Alpha) GlyphMask {
	if alpha == nil { return nil }

	// NOTICE: since ebiten doesn't have good support for alpha images,
	//         this is quite a pain, but not much we can do from here.
	rgba   := image.NewRGBA(alpha.Rect)
	pixels := rgba.Pix
	index  := 0
	for _, value := range alpha.Pix {
		// NOTE: we could actually skip when value == 0, no? benchmark?
		//       or maybe amortizing this image is a better idea..?
		//       like, I could have a timeout, but that seems so overkill.
		//       but having no release method seems dirty too. I could
		//       also keep it throughout a single draw OP, but... that may
		//       or may not work well, hard to say. I also need to pass it
		//       and control it manually. can't share between renderers.
		//       actually, passing a buffer to this function is not crazy.
		pixels[index + 0] = value
		pixels[index + 1] = value
		pixels[index + 2] = value
		pixels[index + 3] = value
		index += 4
	}
	opts := ebiten.NewImageFromImageOptions{ PreserveBounds: true }
	return ebiten.NewImageFromImageWithOptions(rgba, &opts)
}
