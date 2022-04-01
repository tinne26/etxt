//go:build gtxt

package main

import "os"
import "log"
import "fmt"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/emask"

import "golang.org/x/image/math/fixed"

// Must be compiled with '-tags gtxt'

// More than an example, this is something I use when debugging effects
// and rasterizers to print mask glyph data directly and be able to
// see it and analyze it.

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
	fauxRast := emask.FauxRasterizer{}
	fauxRast.SetSkewFactor(-1.0)
	fauxRast.SetExtraWidth(0)
	renderer := etxt.NewRenderer(&fauxRast)
	renderer.SetSizePx(36)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.Traverse("d", fixed.P(0, 0),
		func(dot fixed.Point26_6, _ rune, glyphIndex etxt.GlyphIndex) {
			mask := renderer.LoadGlyphMask(glyphIndex, dot)
			n   := 0
			row := 0
			for n < len(mask.Pix) {
				fmt.Printf("%03d: %03v\n", row, mask.Pix[n : n + mask.Stride])
				n += mask.Stride
				row += 1
			}
		})
	fmt.Print("Program exited successfully.\n")
}
