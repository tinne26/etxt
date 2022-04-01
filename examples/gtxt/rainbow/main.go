//go:build gtxt

package main

import "os"
import "image"
import "image/color"
import "image/png"
import "path/filepath"
import "log"
import "fmt"

import "github.com/tinne26/etxt"

import "golang.org/x/image/math/fixed"

// Must be compiled with '-tags gtxt'

// NOTE: see gtxt/mirror if you want a more advanced example of drawing each
//       character individually. This one uses the renderer's DefaultDrawFunc,
//       so all the heavy lifting is already done.

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	font, fontName, err := etxt.ParseFontFrom(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	// (we omit the cache as we don't reuse any letters anyway...)
	renderer := etxt.NewStdRenderer()
	renderer.SetSizePx(48)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)

	// create target image and fill it with a white to black gradient
	outImage := image.NewRGBA(image.Rect(0, 0, 256, 64))
	for y := 0; y < 64; y++ {
		lvl := 255 - uint8(y*8)
		if y >= 32 { lvl = 255 - lvl }
		for x := 0; x < 256; x++ {
			outImage.Set(x, y, color.RGBA{lvl, lvl, lvl, 255})
		}
	}

	// set target and prepare rainbow colors
	renderer.SetTarget(outImage)
	colors := []color.RGBA {
		color.RGBA{ R: 255, G:   0, B:   0, A: 255 }, // red
		color.RGBA{ R: 255, G: 165, B:   0, A: 255 }, // orange
		color.RGBA{ R: 255, G: 255, B:   0, A: 255 }, // yellow
		color.RGBA{ R:   0, G: 255, B:   0, A: 255 }, // green
		color.RGBA{ R:   0, G:   0, B: 255, A: 255 }, // blue
		color.RGBA{ R:  75, G:   0, B: 130, A: 255 }, // indigo
		color.RGBA{ R: 238, G: 130, B: 238, A: 255 }, // violet
	}

	// draw each letter with a different color
	colorIndex := 0
	renderer.Traverse("RAINBOW", fixed.P(128, 32),
		func(dot fixed.Point26_6, _ rune, glyphIndex etxt.GlyphIndex) {
			renderer.SetColor(colors[colorIndex])
			mask := renderer.LoadGlyphMask(glyphIndex, dot)
			renderer.DefaultDrawFunc(dot, mask, glyphIndex)
			colorIndex += 1
		})

	// store result as png
	filename, err := filepath.Abs("gtxt_rainbow.png")
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
