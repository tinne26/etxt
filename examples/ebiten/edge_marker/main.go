package main

import "os"
import "log"
import "fmt"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/emask"

// For development of the experimental EdgeMarker rasterizer.

const MainText = "edge marker\nwork in progress\nwear your helmet"

type Game struct { txtRenderer *etxt.Renderer }

func (self *Game) Layout(w int, h int) (int, int) { return w, h }
func (self *Game) Update() error { return nil }
func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw text
	w, h := screen.Size()
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.Draw(MainText, w/2, h/2)
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

	// the experimental rasterizer
	edgeRast := &emask.EdgeMarkerRasterizer{}
	edgeRast.SetMode(emask.EdgeRastFill) // EdgeRastRaw also works
	//edgeRast.Marker().SetMaxCurveSplits(0) // fun at large sizes

	// create and configure renderer
	renderer := etxt.NewRenderer(edgeRast)
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(82)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/edge_marker")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game { renderer })
	if err != nil { log.Fatal(err) }
}
