//go:build gtxt

package main

import (
	"fmt"
	"image"
	"log"
	"os"

	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/etxt/mask"
	"golang.org/x/image/font/sfnt"
)

// More than an example, this is something I use when debugging effects
// and rasterizers to print mask glyph data directly and be able to
// see it and analyze it.

const GlyphToDebug = 'Q'

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
	// (notice that we don't set a cache, no need for a single glyph)
	const FontSize = 10
	renderer := etxt.NewRenderer()
	renderer.SetSize(FontSize)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)
	renderer.Fract().SetHorzQuantization(etxt.QtFull)
	renderer.Fract().SetVertQuantization(etxt.QtFull)

	// set a custom rasterizer that we want to debug
	fauxRast := mask.FauxRasterizer{}
	//fauxRast.SetSkewFactor(-0.3)
	fauxRast.SetExtraWidth(+0.0)
	renderer.Glyph().SetRasterizer(&fauxRast)

	// set the debugging draw function
	renderer.Glyph().SetDrawFunc(
		func(_ etxt.Target, glyphIndex sfnt.GlyphIndex, position fract.Point) {
			mask := renderer.Glyph().LoadMask(glyphIndex, position)
			bounds := mask.Bounds()
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				fmt.Printf("%04d: [ ", y)
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					_, _, _, a := mask.At(x, y).RGBA()
					fmt.Printf("%03d ", a>>8)
				}
				fmt.Printf("]\n")
			}
		})

	// create a target image big enough. while it's not technically used on
	// our custom debugging function, etxt would panic on a nil image or may
	// optimize the draw away if the target is empty or non-intersecting
	target := image.NewRGBA(image.Rect(0, 0, FontSize*2, FontSize*2))

	// draw the glyph to debug to hit the custom debug draw function
	renderer.Draw(target, string(GlyphToDebug), FontSize, FontSize)
	fmt.Print("Program exited successfully.\n")
}
