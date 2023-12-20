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

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"

// Must be compiled with '-tags gtxt'

// This example shows how to draw raw glyphs by encoding them
// inside a Twine object. If you use a font with support for
// complex scripts, some of the glyphs that are drawn may not have
// a corresponding unicode code point and therefore couldn't be
// drawn with the regular etxt.Renderer.Draw() function.

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

	// get the number of glyphs in the font
	numGlyphs := sfntFont.NumGlyphs()
	if numGlyphs <= 0 { log.Fatal("No glyphs found in the font.") }

	// create and configure renderer (no glyphs are likely to
	// be repeated, so we won't bother setting a cache here)
	renderer := etxt.NewRenderer()
	renderer.SetSize(24)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create some text adding arbitrary glyphs to a twine
	var twine etxt.Twine
	rand.Seed(time.Now().UnixNano()) // unnecessary in >=go1.20
	for i := 0; i < 32; i++ {
		twine.AddGlyph(sfnt.GlyphIndex(rand.Intn(numGlyphs)))
	}

	// create target image and fill it with white
	lineHeight := int(renderer.Utils().GetLineHeight()*1.3)
	width := renderer.Twine().Measure(twine).IntWidth() + 24 // add some margin
	outImage := image.NewRGBA(image.Rect(0, 0, width, lineHeight))
	for i := 0; i < width*lineHeight*4; i++ { outImage.Pix[i] = 255 }

	// add glyphs to twine and draw
	
	renderer.Twine().Draw(outImage, twine, width/2, lineHeight/2)

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
