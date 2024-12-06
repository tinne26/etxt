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

// NOTE: see gtxt/mirror if you want a more advanced example of drawing each
//       glyph mask in a custom way. This one uses the default glyphs masks,
//       so all the heavy lifting is already done.

const Text = "RAINBOW" // colors will repeat every 7 letters

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
	// (we omit the cache as we don't reuse any letters...)
	renderer := etxt.NewRenderer()
	renderer.SetSize(48)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)

	// create target image and fill it with a white to black gradient
	width := renderer.Measure(Text).IntWidth() + 24
	outImage := image.NewRGBA(image.Rect(0, 0, width, 64))
	for y := 0; y < 64; y++ {
		lvl := 255 - uint8(y*8)
		if y >= 32 {
			lvl = 255 - lvl
		}
		for x := 0; x < width; x++ {
			outImage.Set(x, y, color.RGBA{lvl, lvl, lvl, 255})
		}
	}

	// prepare rainbow colors
	colors := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},     // red
		{R: 255, G: 165, B: 0, A: 255},   // orange
		{R: 255, G: 255, B: 0, A: 255},   // yellow
		{R: 0, G: 255, B: 0, A: 255},     // green
		{R: 0, G: 0, B: 255, A: 255},     // blue
		{R: 75, G: 0, B: 130, A: 255},    // indigo
		{R: 238, G: 130, B: 238, A: 255}, // violet
	}

	// set custom rendering function
	colorIndex := 0
	renderer.Glyph().SetDrawFunc(
		func(target etxt.Target, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
			renderer.SetColor(colors[colorIndex%7])
			mask := renderer.Glyph().LoadMask(glyphIndex, origin)
			renderer.Glyph().DrawMask(target, mask, origin)
			colorIndex += 1
		})

	// draw the text
	renderer.Draw(outImage, Text, width/2, 32)

	// store result as png
	filename, err := filepath.Abs("gtxt_rainbow.png")
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
