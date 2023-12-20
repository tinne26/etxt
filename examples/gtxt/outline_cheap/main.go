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

// This example draws text with a cheap and simple outline, made by
// repeatedly drawing text slightly shifted to the left, right, up
// and down. For higher quality outlines, see the OutlineRasterizer
// instead and the gtxt/outline example.
//
// If you want a more advanced example on how to draw glyphs individually,
// check gtxt/mirror instead. This example uses the renderer's DefaultDrawFunc,
// so it doesn't get into the grittiest details.

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
	renderer.SetSize(36)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create target image and fill it with white
	w, h := 312, 64
	outImage := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h*4; i++ { outImage.Pix[i] = 255 }

	// The key idea is to draw text repeatedly, slightly shifted
	// to the left, right, up, down... and finally draw the middle.
	// We could also do this with separate Draw() calls, but using
	// a custom function is a more general and tweakable approach.
	//
	// We will still draw the main text on a separate call afterwards
	// in order to avoid the background of a letter being overlayed
	// on top of a previously drawn letter (won't happen on most fonts
   // or sizes or glyph sequences, but it's possible in some cases).
	renderer.Glyph().SetDrawFunc(
		func(target etxt.Target, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
			mask := renderer.Glyph().LoadMask(glyphIndex, origin)
			origin.X -= fract.One // shift left
			renderer.Glyph().DrawMask(target, mask, origin)
			origin.X += fract.One*2 // shift right
			renderer.Glyph().DrawMask(target, mask, origin)
			origin.X -= fract.One // restore X to center
			origin.Y -= fract.One // shift up
			renderer.Glyph().DrawMask(target, mask, origin)
			origin.Y += fract.One*2 // shift down
			renderer.Glyph().DrawMask(target, mask, origin)
		})
	renderer.Draw(outImage, "Cheap Outline!", w/2, h/2)

	// finally draw the main text. you can try different colors, but
	// white makes it look like there's only outline, so that's cool.
	renderer.SetColor(color.RGBA{255, 255, 255, 255})
	renderer.Glyph().SetDrawFunc(nil) // restore default draw function
	renderer.Draw(outImage, "Cheap Outline!", 156, 32)

	// store result as png
	filename, err := filepath.Abs("gtxt_outline_cheap.png")
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
