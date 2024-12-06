package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
)

// This example is only for testing what happens when glyphs are
// missing if a suitable fallback handler is set.
//
// You can run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/miss_handler@latest path/to/font.ttf

const Content = "We have àccëntš, we have the ру́сский алфави́т, we have japanese 漢字."

type Game struct {
	text *etxt.Renderer
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth := int(math.Ceil(float64(winWidth) * scale))
	canvasHeight := int(math.Ceil(float64(winHeight) * scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	return nil
}

const NumContentTypes = 5

func (self *Game) Draw(canvas *ebiten.Image) {
	canvas.Fill(color.RGBA{3, 2, 0, 255})
	bounds := canvas.Bounds()
	w := bounds.Dx()
	x, y := bounds.Min.X+w/2, bounds.Min.Y+bounds.Dy()/2

	self.text.SetSize(18)
	self.text.SetColor(color.RGBA{255, 255, 255, 255})
	self.text.SetAlign(etxt.Center)
	self.text.DrawWithWrap(canvas, Content, x, y, w-w/4)
}

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
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetFont(sfntFont)
	renderer.SetColor(color.RGBA{128, 128, 128, 255})
	renderer.SetAlign(etxt.LastBaseline | etxt.Left)
	renderer.SetSize(16)

	// miss handler
	renderer.Glyph().SetMissHandler(etxt.OnMissNotdef)
	// glyph := renderer.Glyph().GetRuneIndex('?')
	// renderer.Glyph().SetMissHandler(func(*sfnt.Font, rune) (sfnt.GlyphIndex, bool) {
	// 	return glyph, false
	// })

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/miss_handler")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game{
		text: renderer,
	})
	if err != nil {
		log.Fatal(err)
	}
}
