//go:build gtxt

package main

import "os"
import "image"
import "image/color"
import "image/png"
import "path/filepath"
import "log"
import "fmt"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"

// Must be compiled with '-tags gtxt'

// An example showcasing how to draw glyphs manually and applying a
// specific pattern effect. The manual glyph drawing part is similar to
// examples/gtxt/mirror.

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetSize(64)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white

	// create target image and fill it with black
	w, h := 360, 64
	outImage := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 3; i < w*h*4; i += 4 { outImage.Pix[i] = 255 }

	// set custom draw func and draw
	renderer.Glyph().SetDrawFunc(
		func(target etxt.Target, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
			mask := renderer.Glyph().LoadMask(glyphIndex, origin)
			drawAsPattern(outImage, mask, origin)
		})
	renderer.Draw(outImage, "PATTERN", 180, 32)

	// store result as png
	filename, err := filepath.Abs("gtxt_pattern.png")
	if err != nil { log.Fatal(err) }
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil { log.Fatal(err) }
	err = png.Encode(file, outImage)
	if err != nil { log.Fatal(err) }
	err = file.Close()
	if err != nil { log.Fatal(err) }
	fmt.Print("Program exited successfully.\n")
}

func drawAsPattern(target *image.RGBA, mask etxt.GlyphMask, origin fract.Point) {
	// to draw a mask into a target, we need to displace it by the
	// current drawing position and be careful with clipping
	srcRect, destRect := getDrawBounds(mask.Rect, target.Bounds(), origin)
	if destRect.Empty() { return } // nothing to draw

	// we now have two rects that are the same size but identify
	// different regions of the mask and target images. we can use
	// them to read from one and draw on the other. yay.

	// we start by creating some helper variables to make iteration
	// through the rects more pleasant
	width    := srcRect.Dx()
	height   := srcRect.Dy()
	srcOffX  := srcRect.Min.X
	srcOffY  := srcRect.Min.Y
	destOffX := destRect.Min.X
	destOffY := destRect.Min.Y

	// iterate the rects and draw!
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// pattern filtering, edit and make your own!
			// e.g:
			// >> (x + y) % 2 == 0
			// >> x % 3 != 2 && y % 3 != 2
			// >> x % 3 == 2 || y % 3 == 2
			// >> x == y
			// >> (width - x) % 5 == y % 5
			// >> (y > height/2) && (x + y) % 2 == 0
			discard := x % 2 != 0 || y % 2 != 0
			if discard { continue }

			// get mask alpha level
			level := mask.AlphaAt(srcOffX + x, srcOffY + y).A
			if level == 0 { continue } // non-filled part of the glyph

			// now we finally can draw to the target
			target.SetRGBA(destOffX + x, destOffY + y, color.RGBA{255, 255, 255, 255})
		}
	}
}

// same as in gtxt/mirror
func getDrawBounds(srcRect, targetRect image.Rectangle, origin fract.Point) (image.Rectangle, image.Rectangle) {
	shift := image.Pt(origin.X.ToIntFloor(), origin.Y.ToIntFloor())
	destRect := targetRect.Intersect(srcRect.Add(shift))
	shift.X, shift.Y = -shift.X, -shift.Y
	return destRect.Add(shift), destRect
}
