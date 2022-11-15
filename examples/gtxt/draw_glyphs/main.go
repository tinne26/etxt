//go:build gtxt

package main

import "os"
import "image"
import "image/color"
import "image/png"
import "path/filepath"
import "log"
import "fmt"
import "time"
import "math/rand"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/eglyr"

// Must be compiled with '-tags gtxt'

// This example shows how to use the eglyr.Renderer by drawing
// random glyphs in the font. If you use a font with support for
// complex scripts, some of the glyphs that are drawn may not have
// a corresponding unicode code point and therefore couldn't be
// drawn with the regular etxt.Renderer.Draw() function.

// Notice also that this example has its own go.mod to add the eglyr
// dependency. This means that if you cloned the repo you won't be
// able to run this example from the etxt folder directly, unlike
// most other examples. You must either use go run from the specific
// program folder or create a go.work file that uses this location:
// >> go work use ./examples/gtxt/draw_glyphs

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

	// get the number of glyphs in the font
	numGlyphs := font.NumGlyphs()
	if numGlyphs <= 0 { log.Fatal("No glyphs found in the font.") }

	// create and configure renderer (no glyphs are likely to
	// be repeated, so we won't bother creating a cache here)
	renderer := eglyr.NewStdRenderer()
	renderer.SetSizePx(24)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create some text using arbitrary glyphs
	rand.Seed(time.Now().UnixNano())
	glyphs := make([]etxt.GlyphIndex, 32)
	for i := 0; i < len(glyphs); i++ {
		glyphs[i] = etxt.GlyphIndex(rand.Intn(numGlyphs))
	}

	// create target image and fill it with white
	width := renderer.SelectionRect(glyphs).Width.Ceil() + 16 // 16px of margin
	outImage := image.NewRGBA(image.Rect(0, 0, width, 42))
	for i := 0; i < width*42*4; i++ { outImage.Pix[i] = 255 }

	// set target and prepare align and draw
	renderer.SetTarget(outImage)
	renderer.Draw(glyphs, width/2, 42/2)

	// store image as png
	filename, err := filepath.Abs("gtxt_draw_glyphs.png")
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
