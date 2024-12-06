//go:build gtxt

package etxt

// --- Context ---
// There are three ways to use etxt:
// - Without build tags and with Ebitengine. Ebitengine is imported and
//   used to draw the glyphs into textures that are sent to the GPU and
//   composed there.
// - With "-tags gtxt", which doesn't import Ebitengine and does the
//   whole glyph mask blitting process manually on the CPU.
// - A third way exists, using both "-tags gtxt" *and* Ebitengine,
//   because Ebitengine's Image type does also conform to the draw.Image
//   interface required to operate under the gtxt mode. This is never
//   recommended for end users of the library, and there are actually
//   some limitations with BlendModes and others. This is only helpful
//   for internal testing of the library.
// So, "ebiten_no" actually means "etxt without importing/depending on
// Ebitengine", but you can still end up using this file with Ebitengine
// programs and the gtxt tag at the same time. It just tends to be a most
// terrible idea.

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/tinne26/etxt/fract"
)

type Target = draw.Image
type GlyphMask = *image.Alpha

type BlendMode uint8

const (
	BlendOver     BlendMode = 0 // glyphs drawn over target (default mode)
	BlendReplace  BlendMode = 1 // glyph mask only (transparent pixels included!)
	BlendAdd      BlendMode = 2 // add colors (black adds nothing, white stays white)
	BlendSub      BlendMode = 3 // subtract colors (black removes nothing) (alpha = target)
	BlendMultiply BlendMode = 4 // multiply % of glyph and target colors and MixOver
	BlendCut      BlendMode = 5 // cut glyph shape hole based on alpha (cutout text)
	BlendHue      BlendMode = 6 // keep highest alpha, blend hues proportionally

	// TODO: many of the modes above will have some trouble with
	//       semi-transparency, I should look more into it.
)

// this doesn't do anything in gtxt, only ebiten needs it
func convertAlphaImageToGlyphMask(i *image.Alpha) GlyphMask { return i }

// Underlying default glyph drawing function for renderers.
// Can be overridden with Renderer.Glyph().SetDrawFunc(...).
func (self *Renderer) defaultDrawFunc(target Target, origin fract.Point, mask GlyphMask) {
	if mask == nil {
		return
	} // spaces and empty glyphs will be nil

	// compute src and target rects within bounds
	targetBounds := target.Bounds()
	srcRect := mask.Rect
	shift := image.Pt(origin.X.ToIntFloor(), origin.Y.ToIntFloor())
	targetRect := targetBounds.Intersect(srcRect.Add(shift))
	if targetRect.Empty() {
		return
	}
	shift.X, shift.Y = -shift.X, -shift.Y
	srcRect = targetRect.Add(shift)

	switch self.state.blendMode {
	case BlendReplace: // ---- source only ----
		self.mixImageInto(mask, target, srcRect, targetRect,
			func(new, _ color.Color) color.Color { return new })
	case BlendOver: // ---- default mixing ----
		self.mixImageInto(mask, target, srcRect, targetRect, blendOverFunc)
	case BlendCut: // ---- remove alpha mode ----
		self.mixImageInto(mask, target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				_, _, _, na := new.RGBA()
				if na == 0 {
					return curr
				}
				cr, cg, cb, ca := curr.RGBA()

				alpha := ca - na
				if alpha < 0 {
					alpha = 0
				}
				return color.RGBA64{
					R: min32As16(cr, alpha),
					G: min32As16(cg, alpha),
					B: min32As16(cb, alpha),
					A: uint16(alpha),
				}
			})
	case BlendMultiply: // ---- multiplicative blending ----
		self.mixImageInto(mask, target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				nr, ng, nb, na := new.RGBA()
				cr, cg, cb, ca := curr.RGBA()
				pureMult := color.RGBA64{
					R: uint16(nr * cr / 0xFFFF),
					G: uint16(ng * cg / 0xFFFF),
					B: uint16(nb * cb / 0xFFFF),
					A: uint16(na * ca / 0xFFFF),
				}
				return blendOverFunc(pureMult, curr)
			})
	case BlendAdd: // --- additive blending ----
		self.mixImageInto(mask, target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				nr, ng, nb, na := new.RGBA()
				if na == 0 {
					return curr
				}
				cr, cg, cb, ca := curr.RGBA()
				return color.RGBA64{
					R: uint16N(nr + cr),
					G: uint16N(ng + cg),
					B: uint16N(nb + cb),
					A: uint16N(na + ca),
				}
			})
	case BlendSub: // --- subtractive blending (only color) ----
		self.mixImageInto(mask, target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				nr, ng, nb, na := new.RGBA()
				if na == 0 {
					return curr
				}
				cr, cg, cb, ca := curr.RGBA()
				return color.RGBA64{
					R: uint32subFloor16(cr, nr),
					G: uint32subFloor16(cg, ng),
					B: uint32subFloor16(cb, nb),
					A: uint16(ca),
				}
			})
	case BlendHue: // ---- max alpha, proportional hue blending ----
		self.mixImageInto(mask, target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				var nr, ng, nb, na uint32 = new.RGBA()
				if na == 0 {
					return curr
				}
				cr, cg, cb, ca := curr.RGBA()
				if ca == 0 {
					return new
				}

				// hue contribution is proportional to alpha.
				// if both alphas are equal, hue contributions are 50/50
				ta := ca + na // alpha sum (total)
				ma := ca      // max alpha
				if na > ca {
					ma = na
				}
				r := (((nr + cr) >> 1) * ma) / (ta >> 1) // shifts prevent overflows
				g := (((ng + cg) >> 1) * ma) / (ta >> 1)
				b := (((nb + cb) >> 1) * ma) / (ta >> 1)
				partial := color.RGBA64{
					R: uint16(r),
					G: uint16(g),
					B: uint16(b),
					A: uint16(ma),
				}
				return blendOverFunc(partial, curr)
			})
	default:
		panic("unexpected blend mode")
	}
}

// All this code is extremely slow due to using a very straightforward
// implementation. Making this faster, though, is not so trivial.
func (self *Renderer) mixImageInto(src GlyphMask, target draw.Image, srcRect, tarRect image.Rectangle, mixFunc func(color.Color, color.Color) color.Color) {
	width := srcRect.Dx()
	height := srcRect.Dy()
	srcOffX := srcRect.Min.X
	srcOffY := srcRect.Min.Y
	tarOffX := tarRect.Min.X
	tarOffY := tarRect.Min.Y

	directColor := self.state.fontColor
	r, g, b, a := directColor.RGBA()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// get mask alpha applied to our main drawing color
			level := src.AlphaAt(srcOffX+x, srcOffY+y).A
			var newColor color.Color
			if level == 0 {
				newColor = color.RGBA{0, 0, 0, 0}
			} else if level == 255 {
				newColor = directColor
			} else {
				newColor = rescaledAlpha(r, g, b, a, level)
			}

			// get target current color and mix
			currColor := target.At(tarOffX+x, tarOffY+y)
			mixColor := mixFunc(newColor, currColor)
			target.Set(tarOffX+x, tarOffY+y, mixColor)
		}
	}
}

func rescaledAlpha(r, g, b, a uint32, alphaFactor uint8) color.Color {
	return color.RGBA64{
		R: uint16((r * uint32(alphaFactor)) / 255),
		G: uint16((g * uint32(alphaFactor)) / 255),
		B: uint16((b * uint32(alphaFactor)) / 255),
		A: uint16((a * uint32(alphaFactor)) / 255),
	}
}

func uint16N(value uint32) uint16 {
	if value > 65535 {
		return 65535
	}
	return uint16(value)
}

func uint32subFloor16(a, b uint32) uint16 {
	if b >= a {
		return 0
	}
	return uint16(a - b)
}

func min32As16(a, b uint32) uint16 {
	if a <= b {
		return uint16(a)
	}
	return uint16(b)
}

// ---- color blending functions ----
func blendOverFunc(new, curr color.Color) color.Color {
	nr, ng, nb, na := new.RGBA()
	if na == 0xFFFF {
		return new
	}
	if na == 0 {
		return curr
	}
	cr, cg, cb, ca := curr.RGBA()
	if ca == 0 {
		return new
	}

	return color.RGBA64{
		R: uint16(nr + (cr*(0xFFFF-na))>>16), // dividing by 0xFFFF is also possible, but
		G: uint16(ng + (cg*(0xFFFF-na))>>16), // we get as much difference with ebitengine
		B: uint16(nb + (cb*(0xFFFF-na))>>16), // as with >> 16, so we prefer going fast
		A: uint16(na + (ca*(0xFFFF-na))>>16),
	}
}
