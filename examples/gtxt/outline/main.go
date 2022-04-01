//go:build gtxt

package main

import "os"
import "image"
import "image/color"
import "image/png"
import "path/filepath"
import "log"
import "fmt"

import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt"

// Must be compiled with '-tags gtxt'

// NOTE: see gtxt/mirror if you want a more advanced example on how to
//       draw glyphs individually. This one uses the renderer's
//       DefaultDrawFunc, so it doesn't get into the grittiest details.

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

	// create cache
	cache := etxt.NewDefaultCache(1024*1024*1024) // 1GB cache

	// create and configure renderer
	renderer := etxt.NewStdRenderer()
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(36)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create target image and fill it with white
	outImage := image.NewRGBA(image.Rect(0, 0, 312, 64))
	for i := 0; i < 312*64*4; i++ { outImage.Pix[i] = 255 }

	// set target and start drawing each character...
	renderer.SetTarget(outImage)

	// --- key ideas for drawing some ~very crude~ outlined text ---
	// The key idea is to draw text repeatedly, slightly shifted
	// to the left, right, up, down... and finally draw the middle.
	// We could so this with separate Draw() calls and the code
	// would be shorter, but hey, we are exploring, so we will take
	// advantage of Traverse to do all the background text in
	// the same place.
	//
	// We will still draw the main text on a separate call afterwards
	// in order to avoid the background of a letter being overlayed
	// on top of a previously drawn letter (won't happen on most fonts
   // or sizes or glyph sequences, but it's possible in some cases).
	//
	// Similar effects could be achieved with shaders or custom
	// rasterizers, but that's enough spamming for today...
	// -------------------------------------------------------------
	renderer.Traverse("Ugly Outline!", fixed.P(156, 32),
		func(dot fixed.Point26_6, _ rune, glyphIndex etxt.GlyphIndex) {
			const DotShift = 1 << 6 // we want to shift the letters 1 pixel
			                        // to create an outline, but since we are
											// using fixed precision numbers with 6
											// bits for the decimal part, we need to
											// apply this shift for the number to be
											// correct in fixed.Int26_6 format.

			mask := renderer.LoadGlyphMask(glyphIndex, dot)
			dot.X -= DotShift // shift left
			renderer.DefaultDrawFunc(dot, mask, glyphIndex)
			dot.X += DotShift*2 // shift right
			renderer.DefaultDrawFunc(dot, mask, glyphIndex)
			dot.X -= DotShift // restore X to center
			dot.Y -= DotShift // shift up
			renderer.DefaultDrawFunc(dot, mask, glyphIndex)
			dot.Y += DotShift*2 // shift down
			renderer.DefaultDrawFunc(dot, mask, glyphIndex)
		})

	// finally draw the main text. you can try different colors, but
	// white makes it look like there's only outline, so that's cool.
	renderer.SetColor(color.RGBA{255, 255, 255, 255})
	renderer.Draw("Ugly Outline!", 156, 32)

	// store result as png
	filename, err := filepath.Abs("gtxt_outline.png")
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
