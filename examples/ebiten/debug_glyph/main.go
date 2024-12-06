package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/etxt/mask"
	"golang.org/x/image/font/sfnt"
)

// This is the Ebitengine version of gtxt/debug_glyph. Not a true example, but
// rather a debug program for when developing custom rasterizers and wanting
// to check the results manually. In the case of Ebitengine, the values have
// to pass through the GPU, and values may vary slightly compared to the gtxt
// version (CPU rendering).

const GlyphToDebug = 'A'

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
	renderer := etxt.NewRenderer()
	renderer.SetSize(18)
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

	// draw the glyph to debug to hit the custom debug draw function
	err = ebiten.RunGame(&Game{text: renderer})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("Program exited successfully.\n")
}

// The game struct is required for ebitengine to initialize the graphical
// command queue, but we are only trying to invoke our custom text rendering
// debug function once and then we terminate right away.
type Game struct {
	text *etxt.Renderer
	done bool
}

func (self *Game) Layout(w, h int) (int, int) { return w, h }
func (self *Game) Update() error {
	if self.done {
		return ebiten.Termination
	}
	return nil
}
func (self *Game) Draw(canvas *ebiten.Image) {
	if self.done {
		return
	}
	bounds := canvas.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	self.text.Draw(canvas, string(GlyphToDebug), w/2, h/2)
	self.done = true
}
