package main

import "os"
import "log"
import "fmt"
import "math"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"

// This example shows how to combine a couple draws and some 
// very basic logic in order to create a simple effect when
// hovering text with the mouse. There are still a few interesting
// details here and there if you are still getting started with
// etxt and Ebitengine, like measuring the text and manipulating
// its fract.Rect, or adjusting the animation based on the display
// scaling for consistent results across different setups.
// You can run this example with:
//   go run github.com/tinne26/etxt/examples/ebiten/hover_shadow@latest path/to/font.ttf

const HoverText = "Hover me please!"

type Game struct {
	text *etxt.Renderer
	focus float64
	canvasWidth int
	canvasHeight int
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	self.canvasWidth  = int(math.Ceil(float64(winWidth)*scale))
	self.canvasHeight = int(math.Ceil(float64(winHeight)*scale))
	return self.canvasWidth, self.canvasHeight
}

func (self *Game) Update() error {
	// calculate target area. you could easily optimize this,
	// but we are being lazy and wasteful... and it's still ok
	targetRect := self.text.Measure(HoverText)
	ox, oy := self.canvasWidth/2, self.canvasHeight/2
	targetRect = targetRect.CenteredAtIntCoords(ox, oy)

	// determine if we are inside or outside the
	// hover area and adjust the "focus" level
	cursorPt := fract.IntsToPoint(ebiten.CursorPosition())
	if targetRect.Contains(cursorPt) {
		self.focus += 0.05
		if self.focus > 1.0 { self.focus = 1.0 }
	} else {
		self.focus -= 0.05
		if self.focus < 0.0 { self.focus = 0.0 }
	}

	return nil
}

func (self *Game) Draw(canvas *ebiten.Image) {
	const MaxOffsetX = 4 // max shadow x offset
	const MaxOffsetY = 4 // max shadow y offset

	// dark background
	canvas.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw text
	if self.focus > 0 {
		self.text.SetColor(color.RGBA{200, 0, 200, 200}) // sharp shadow
		scale := ebiten.DeviceScaleFactor()
		hx := self.canvasWidth/2  + int(self.focus*MaxOffsetX*scale)
		hy := self.canvasHeight/2 + int(self.focus*MaxOffsetY*scale)
		self.text.Draw(canvas, HoverText, hx, hy)
	}

	self.text.SetColor(color.RGBA{255, 255, 255, 255}) // main color
	self.text.Draw(canvas, HoverText, self.canvasWidth/2, self.canvasHeight/2)
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
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetSize(64)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/hover_shadow")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game{ text: renderer })
	if err != nil { log.Fatal(err) }
}
