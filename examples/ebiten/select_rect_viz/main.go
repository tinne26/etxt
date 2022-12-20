package main

import "image"
import "os"
import "image/color"
import "log"
import "fmt"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt"

// This example allows you to interactively write text in an Ebitengine
// program and see how the SelectionRect for the text changes. You
// can use backspace to remove characters, and enter to create line
// breaks.

type Game struct {
	txtRenderer *etxt.Renderer
	sinceLastSpecialKey int
	text []rune
}

func (self *Game) Layout(w int, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	return int(float64(w)*scale), int(float64(h)*scale)
}
func (self *Game) Update() error {
	backspacePressed := ebiten.IsKeyPressed(ebiten.KeyBackspace)
	enterPressed     := ebiten.IsKeyPressed(ebiten.KeyEnter)

	if backspacePressed && self.sinceLastSpecialKey >= 7 && len(self.text) >= 1 {
		self.sinceLastSpecialKey = 0
		self.text = self.text[0 : len(self.text) - 1]
	} else if enterPressed && self.sinceLastSpecialKey >= 20 {
		self.sinceLastSpecialKey = 0
		self.text = append(self.text, '\n')
	} else {
		self.sinceLastSpecialKey += 1
		self.text = ebiten.AppendInputChars(self.text)
	}

	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 2, 1, 0, 255 })

	// draw text's selection rect
	x, y := 8, 8
	rect := self.txtRenderer.SelectionRect(string(self.text)).ImageRect()
	rectImg := screen.SubImage(rect.Add(image.Pt(x, y))).(*ebiten.Image)
	rectImg.Fill(color.RGBA{ 8, 72, 88, 255 })

	// draw text
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.Draw(string(self.text), x, y)

	// draw fps and other info for fun
	self.txtRenderer.SetAlign(etxt.Bottom, etxt.Right)
	w, h := screen.Size()
	info := fmt.Sprintf("%d glyphs - %.2fFPS", len(self.text), ebiten.CurrentFPS())
	self.txtRenderer.Draw(info, w - 2, h - 2)
	self.txtRenderer.SetAlign(etxt.Top, etxt.Left)
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
	scale := ebiten.DeviceScaleFactor()
	renderer := etxt.NewStdRenderer()
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(int(18*scale))
	renderer.SetFont(font)
	renderer.SetAlign(etxt.Top, etxt.Left)
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/select_rect_viz")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game { renderer, 0, []rune("Interactive text") })
	if err != nil { log.Fatal(err) }
}
