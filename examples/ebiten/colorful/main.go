package main

import "os"
import "log"
import "fmt"
import "math"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt"

type Game struct {
	txtRenderer *etxt.Renderer
	redSrc   float64
	greenSrc float64
	blueSrc  float64
}

func (self *Game) Layout(w int, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	return int(math.Ceil(float64(w)*scale)), int(math.Ceil(float64(h)*scale))
}
func (self *Game) Update() error {
	// progressively change the values to use in Draw to derive colors,
	// at different speeds each one
	self.redSrc   -= 0.0202
	self.greenSrc -= 0.0168
	self.blueSrc  -= 0.0227
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })
	offset := 0.0

	// draw text
	const MainText = "Colorful!\nWonderful!"
	w, h := screen.Size()
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.Traverse(MainText, fixed.P(w/2, h/2),
		func(dot fixed.Point26_6, codePoint rune, glyphIndex etxt.GlyphIndex) {
			if codePoint == '\n' { return } // skip line breaks

			// derive the color for the current letter from the Src values on
			// each color channel, the current offset, and the sine function
			r := (math.Sin(self.redSrc + offset) + 1.0)/2.0
			g := (math.Sin(self.greenSrc + offset) + 1.0)/2.0
			b := (math.Sin(self.blueSrc + offset) + 1.0)/2.0
			self.txtRenderer.SetColor(color.RGBA{uint8(r*255), uint8(g*255), uint8(b*255), 255})

			// draw the glyph mask
			mask := self.txtRenderer.LoadGlyphMask(glyphIndex, dot)
			self.txtRenderer.DefaultDrawFunc(dot, mask, glyphIndex)

			// increase offset to apply to the next letters
			offset += 0.15
		})
}

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
	renderer.SetSizePx(int(64*ebiten.DeviceScaleFactor()))
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/colorful")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game { renderer, -5.54, -4.3, -6.4 })
	if err != nil { log.Fatal(err) }
}
