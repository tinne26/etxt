//go:build gtxt

package etxt

import "image"
import "image/draw"
import "image/color"
import "golang.org/x/image/math/fixed"

type TargetImage = draw.Image
type GlyphMask = *image.Alpha

type BlendMode uint8

const (
	BlendOver       MixMode = 0 // glyphs drawn over target (default mode)
	BlendReplace    MixMode = 1 // glyph mask only (transparent pixels included!)
	BlendAdd        MixMode = 2 // add colors (black adds nothing, white stays white)
	BlendSub        MixMode = 3 // subtract colors (black removes nothing) (alpha = target)
	BlendMultiply   MixMode = 4 // multiply % of glyph and target colors and MixOver
	BlendCut        MixMode = 5 // cut glyph shape hole based on alpha (cutout text)
	BlendFiftyFifty MixMode = 6 // mix glyph and target hues 50%-50% and BlendOver

	// TODO: many of the modes above will have some trouble with
	//       semi-transparency, I should look more into it.
)

// this doesn't do anything in gtxt, only ebiten needs it
func convertAlphaImageToGlyphMask(i *image.Alpha) GlyphMask { return i }

// TODO: just remove glyph index from default draw func if unused everywhere.
//       Maybe replace with ColorM? like... that would seem much more
//       reasonable for traverse. we are doing a lot of color conversion each
//       time. hmmmm... at least for ebitengine. 

// The default glyph drawing function used in renderers. Do not confuse with
// the main [Renderer.Draw]() function. DefaultDrawFunc is a low level function,
// rarely necessary except when paired with [Renderer.Traverse]*() operations.
func (self *Renderer) DefaultDrawFunc(origin fixed.Point26_6, mask GlyphMask, _ GlyphIndex) {
	if mask == nil { return } // spaces and empty glyphs will be nil

	// compute src and target rects within bounds
	targetBounds := self.target.Bounds()
	srcRect := mask.Rect
	shift := image.Pt(origin.X.Floor(), origin.Y.Floor())
	targetRect := targetBounds.Intersect(srcRect.Add(shift))
	if targetRect.Empty() { return }
	shift.X, shift.Y = -shift.X, -shift.Y
	srcRect = targetRect.Add(shift)

	switch self.blendMode {
	case BlendReplace: // ---- source only ----
		self.mixImageInto(mask, self.target, srcRect, targetRect,
			func(new, _ color.Color) color.Color { return new })
	case BlendOver: // ---- default mixing ----
		self.mixImageInto(mask, self.target, srcRect, targetRect, blendOverFunc)
	case BlendCut: // ---- remove alpha mode ----
		self.mixImageInto(mask, self.target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				_, _, _, na := new.RGBA()
				if na == 0 { return curr }
				cr, cg, cb, ca := curr.RGBA()

				alpha := ca - na
				if alpha < 0 { alpha = 0 }
				return color.RGBA64 {
					R: min32As16(cr, alpha),
					G: min32As16(cg, alpha),
					B: min32As16(cb, alpha),
					A: uint16(alpha),
				}
			})
	case BlendMultiply: // ---- multiplicative blending ----
		self.mixImageInto(mask, self.target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				nr, ng, nb, na := new.RGBA()
				cr, cg, cb, ca := curr.RGBA()
				pureMult := color.RGBA64 {
					R: uint16(nr*cr/0xFFFF),
					G: uint16(ng*cg/0xFFFF),
					B: uint16(nb*cb/0xFFFF),
					A: uint16(na*ca/0xFFFF),
				}
				return blendOverFunc(pureMult, curr)
			})
	case BlendAdd: // --- additive blending ----
		self.mixImageInto(mask, self.target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				nr, ng, nb, na := new.RGBA()
				if na == 0 { return curr }
				cr, cg, cb, ca := curr.RGBA()
				return color.RGBA64 {
					R: uint16N(nr + cr),
					G: uint16N(ng + cg),
					B: uint16N(nb + cb),
					A: uint16N(na + ca),
				}
			})
	case BlendSub: // --- subtractive blending (only color) ----
		self.mixImageInto(mask, self.target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				nr, ng, nb, na := new.RGBA()
				if na == 0 { return curr }
				cr, cg, cb, ca := curr.RGBA()
				return color.RGBA64 {
					R: uint32subFloor16(cr, nr),
					G: uint32subFloor16(cg, ng),
					B: uint32subFloor16(cb, nb),
					A: uint16(ca),
				}
			})
	case BlendFiftyFifty: // ---- 50%-50% hue blending ----
		self.mixImageInto(mask, self.target, srcRect, targetRect,
			func(new, curr color.Color) color.Color {
				var nr, ng, nb, na uint32
				nrgba, isNrgba := new.(color.NRGBA64)
				if isNrgba {
					nr, ng, nb, na = uint32(nrgba.R), uint32(nrgba.G), uint32(nrgba.B), uint32(nrgba.A)
				} else {
					nr, ng, nb, na = new.RGBA()
				}
				if na == 0 { return curr }
				if !isNrgba && na != 65535 { panic("broken assumptions") }
				cr, cg, cb, ca := curr.RGBA()
				alphaSum := na + ca
				if alphaSum > 65535 { alphaSum  = 65535 }
				partial := color.NRGBA64 {
					R: uint16((nr + cr)/2),
					G: uint16((ng + cg)/2),
					B: uint16((nb + cb)/2),
					A: uint16(na),
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
	width  := srcRect.Dx()
	height := srcRect.Dy()
	srcOffX := srcRect.Min.X
	srcOffY := srcRect.Min.Y
	tarOffX := tarRect.Min.X
	tarOffY := tarRect.Min.Y

	r, g, b, a := self.mainColor.RGBA()
	transColor := color.RGBA64{0, 0, 0, 0}
	directColor := color.RGBA64 {
		R: uint16(r),
		G: uint16(g),
		B: uint16(b),
		A: uint16(a),
	}
	nrgba := color.NRGBA64 {
		R: uint16(r),
		G: uint16(g),
		B: uint16(b),
		A: 0,
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// get mask alpha applied to our main drawing color
			level := src.AlphaAt(srcOffX + x, srcOffY + y).A
			var newColor color.Color
			if level == 0 {
				newColor = transColor
			} else if level == 255 {
				newColor = directColor
			} else {
				nrgba.A = uint16((a*uint32(level))/255)
				newColor = nrgba
			}

			// get target current color and mix
			currColor := target.At(tarOffX + x, tarOffY + y)
			mixColor  := mixFunc(newColor, currColor)
			target.Set(tarOffX + x, tarOffY + y, mixColor)
		}
	}
}

func uint16N(value uint32) uint16 {
	if value > 65535 { return 65535 }
	return uint16(value)
}

func uint32subFloor16(a, b uint32) uint16 {
	if b >= a { return 0 }
	return uint16(a - b)
}

func min32As16(a, b uint32) uint16 {
	if a <= b { return uint16(a) }
	return uint16(b)
}


// ---- color blending functions ----
func blendOverFunc(new, curr color.Color) color.Color {
	nr, ng, nb, na := new.RGBA()
	if na == 0xFFFF { return new }
	if na == 0      { return curr }
	cr, cg, cb, ca := curr.RGBA()
	if ca == 0      { return new }

	return color.RGBA64 {
		R: uint16N((nr*0xFFFF + cr*(0xFFFF - na))/0xFFFF),
		G: uint16N((ng*0xFFFF + cg*(0xFFFF - na))/0xFFFF),
		B: uint16N((nb*0xFFFF + cb*(0xFFFF - na))/0xFFFF),
		A: uint16N((na*0xFFFF + ca*(0xFFFF - na))/0xFFFF),
	}
}
