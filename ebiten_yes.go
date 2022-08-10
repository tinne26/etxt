//go:build !gtxt

package etxt

import "image"
import "image/color"

import "golang.org/x/image/math/fixed"
import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt/internal"

// Alias to allow compiling the package without Ebitengine (gtxt version).
//
// Without Ebitengine, TargetImage defaults to [image/draw.Image].
type TargetImage = *ebiten.Image

// Mix modes specify how to compose colors when drawing glyphs
// on the renderer's target:
//  - Without Ebitengine, the mix modes can be MixOver, MixReplace,
//    MixAdd, MixSub, MixMultiply, MixCut and MixFiftyFifty.
//  - With Ebitengine, mix modes are Ebitengine's composite modes.
//
// I only ever change mix modes to make cutout text, but there's a
// lot of weird people out there, what can I say.
type MixMode = ebiten.CompositeMode
const defaultMixMode = ebiten.CompositeModeSourceOver

// The default glyph drawing function used in renderers. Do not confuse with
// the main [Renderer.Draw]() function. DefaultDrawFunc is a low level function,
// rarely necessary except when paired with [Renderer.Traverse]*() operations.
func (self *Renderer) DefaultDrawFunc(dot fixed.Point26_6, mask GlyphMask, _ GlyphIndex) {
	if mask == nil { return } // spaces and empty glyphs will be nil

	// TODO: switch to DrawTriangles to reduce overhead?
	// DrawTriangles(vertices []Vertex, indices []uint16, img *Image, options *DrawTrianglesOptions)
	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(dot.X.Floor() + mask.XOffset), float64(dot.Y.Floor() + mask.YOffset))
	opts.ColorM.Scale(colorToFloat64(self.mainColor))
	opts.CompositeMode = self.mixMode
	self.target.DrawImage(mask.Image, &opts)
}

// Convert a color to its float64 [0, 1.0] components.
// This could actually be memorized to make DefaultDrawFunc work better
// in most cases, but I don't know if it's worth the extra complexity.
func colorToFloat64(subject color.Color) (float64, float64, float64, float64) {
	rgbaColor, isRGBA := subject.(color.RGBA)
	if isRGBA {
		r, g, b, a := rgbaColor.R, rgbaColor.G, rgbaColor.B, rgbaColor.A
		return float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255
	} else {
		r, g, b, a := subject.RGBA()
		return float64(r)/65535, float64(g)/65535, float64(b)/65535, float64(a)/65535
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
		pixels[index + 0] = value
		pixels[index + 1] = value
		pixels[index + 2] = value
		pixels[index + 3] = value
		index += 4
	}
	return &internal.EbitenGlyphMask{
		Image  : ebiten.NewImageFromImage(rgba),
		XOffset: alpha.Rect.Min.X,
		YOffset: alpha.Rect.Min.Y,
	}
}
