package main

import "os"
import "log"
import "fmt"
import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt"
import "golang.org/x/image/math/fixed"

const HoverText = "Hover me please!"

type Game struct {
	txtRenderer *etxt.Renderer
	focus float64
}

func (self *Game) Layout(w int, h int) (int, int) { return w, h }
func (self *Game) Update() error {
	// calculate target area. in general you don't need to recalculate
	// this at every frame, but we are being lazy and wasteful here
	targetArea := self.txtRenderer.SelectionRect(HoverText)
	w, h := ebiten.WindowSize()
	tw, th := targetArea.WidthCeil(), targetArea.HeightCeil()
	tRect := image.Rect(w/2 - tw/2, h/2 - th/2, w/2 + tw/2, h/2 + th/2)

	// determine if we are inside or outside the hover
	// area and adjust the "focus" level
	if image.Pt(ebiten.CursorPosition()).In(tRect) {
		self.focus += 0.06
		if self.focus > 1.0 { self.focus = 1.0 }
	} else {
		self.focus -= 0.06
		if self.focus < 0.0 { self.focus = 0.0 }
	}
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	const MaxOffsetX = 4 // max shadow x offset
	const MaxOffsetY = 4 // max shadow y offset

	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw text
	w, h := screen.Size()
	self.txtRenderer.SetTarget(screen)
	if self.focus > 0 {
		self.txtRenderer.SetColor(color.RGBA{255, 0, 255, 128}) // sharp shadow
		hx := fixed.Int26_6((w/2)*64) + fixed.Int26_6(self.focus*MaxOffsetX*64)
		hy := fixed.Int26_6((h/2)*64) + fixed.Int26_6(self.focus*MaxOffsetY*64)
		self.txtRenderer.DrawFract(HoverText, hx, hy)
	}

	self.txtRenderer.SetColor(color.RGBA{255, 255, 255, 255}) // main color
	self.txtRenderer.Draw(HoverText, w/2, h/2)
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
	renderer.SetSizePx(64)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/hover_shadow")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game { renderer, 0.0 })
	if err != nil { log.Fatal(err) }
}
