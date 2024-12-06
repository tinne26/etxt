//go:build gtxt

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
	"github.com/tinne26/etxt/fract"
	"golang.org/x/image/font/sfnt"
)

// Must be compiled with '-tags gtxt'

const fontSize = 48

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetSize(fontSize)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Baseline | etxt.HorzCenter)
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white

	// create target image and fill it with black
	outImage := image.NewRGBA(image.Rect(0, 0, 256, 128))
	for i := 3; i < 256*128*4; i += 4 {
		outImage.Pix[i] = 255
	}

	// configure a custom drawing function
	renderer.Glyph().SetDrawFunc(
		func(target etxt.Target, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
			// draw the "mirrored" glyph manually *first*, so if there's
			// any overlap with the main glyph (because we are using a rather
			// raw and basic method), the main glyph still gets drawn on top
			mask := renderer.Glyph().LoadMask(glyphIndex, origin)
			customMirroredDraw(outImage, mask, origin)

			// draw the normal letter now
			renderer.Glyph().DrawMask(target, mask, origin)
		})

	// draw using our custom function
	renderer.Draw(outImage, "Mirror...?", 128, 64)

	// store result as png
	filename, err := filepath.Abs("gtxt_mirror.png")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = png.Encode(file, outImage)
	if err != nil {
		log.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("Program exited successfully.\n")
}

// This is the hardcore part of this program. We will use the mask to
// manually draw into the target, applying the given dot drawing position
// and flipping the glyph and stuff.
func customMirroredDraw(target *image.RGBA, mask etxt.GlyphMask, origin fract.Point) {
	// to draw a mask into a target, we need to displace it by the
	// current dot (drawing position) and be careful with clipping
	srcRect, destRect := getDrawBounds(mask.Rect, target.Bounds(), origin)
	if destRect.Empty() {
		return
	} // nothing to draw

	// the destRect bounds are not appropriate here, since we want them
	// to be mirrored. we could have done this in a single function, but
	// the getDrawBounds function can be useful for you in other cases too,
	// and this way we don't mix too much stuff in a single place.
	// ...this also makes this code incorrect under some clipping cases,
	//    but don't worry about it, we will just panic :D
	yFlippingPoint := origin.Y.ToIntFloor()
	above := yFlippingPoint - destRect.Min.Y
	below := destRect.Max.Y - yFlippingPoint
	if below < 0 {
		below = -below
	} // take the absolute value
	shift := above - below
	destRect = destRect.Add(image.Pt(0, shift))
	clipped := target.Bounds().Intersect(destRect)
	if clipped.Dy() != destRect.Dy() {
		msg := "we panic because our code is weak. Here we would have to "
		msg += "re-adjust the source (mask) rect too, but I'm too lazy and "
		msg += "this doesn't happen if you keep reasonable text and target "
		msg += "sizes"
		panic(msg)
	}

	// we now have two rects that are the same size but identify
	// different regions of the mask and target images. we can use
	// them to read from one and draw on the other. yay.

	// we start by creating some helper variables to make iteration
	// through the rects more pleasant
	width := srcRect.Dx()
	height := srcRect.Dy()
	srcOffX := srcRect.Min.X
	srcOffY := srcRect.Min.Y
	destOffX := destRect.Min.X
	destOffY := destRect.Max.Y // (using max for vertical inversion)

	// iterate the rects and draw!
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// get mask alpha level
			level := mask.AlphaAt(srcOffX+x, srcOffY+y).A
			if level == 0 {
				continue
			} // non-filled part of the glyph

			// actually, I also want to make the mirrored image fade out
			// slightly, so let's apply attenuation based on the current y
			attenuationFactor := float64(y) / float64(height)
			attenuationFactor *= 0.76

			// and let's add some noise too, why not...
			noise := rand.Float64() * 70
			flevel := float64(level)
			if flevel <= noise {
				noise = 0
			}
			level = uint8((flevel - noise) * attenuationFactor)

			// now we finally can draw to the target
			color := color.RGBA{level, level, level, 255} // some shade of gray
			target.SetRGBA(destOffX+x, destOffY-y-1, color)
		}
	}
}

// When you have to draw a mask into a target, you need to displace it
// based on the current drawing position and clip the resulting rect
// if it goes out of the target. It's a bit tricky, so here's this nice
// function that deals with it for you. You can reuse it for your own
// code any time you need it. I even considered putting some of these
// trickier functions in a subpackage, but copying is good enough.
func getDrawBounds(srcRect, targetRect image.Rectangle, origin fract.Point) (image.Rectangle, image.Rectangle) {
	shift := image.Pt(origin.X.ToIntFloor(), origin.Y.ToIntFloor())
	destRect := targetRect.Intersect(srcRect.Add(shift))
	shift.X, shift.Y = -shift.X, -shift.Y
	return destRect.Add(shift), destRect
}
