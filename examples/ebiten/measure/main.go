package main

import "os"
import "log"
import "fmt"
import "math"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"

// This example allows you to interactively write text in an Ebitengine
// program and see how the measurement rect for the text changes. You
// can use backspace to remove characters and enter to create line
// breaks. You can run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/measure@latest path/to/font.ttf

type Game struct {
	text *etxt.Renderer
	sinceLastSpecialKey int // to control a repeat effect for backspace and enter
	content []rune // not very efficient, but AppendInputChars uses runes
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	backspacePressed := ebiten.IsKeyPressed(ebiten.KeyBackspace)
	enterPressed     := ebiten.IsKeyPressed(ebiten.KeyEnter)

	if backspacePressed && self.sinceLastSpecialKey >= 7 && len(self.content) >= 1 {
		self.sinceLastSpecialKey = 0
		self.content = self.content[0 : len(self.content) - 1]
	} else if enterPressed && self.sinceLastSpecialKey >= 20 {
		self.sinceLastSpecialKey = 0
		self.content = append(self.content, '\n')
	} else {
		self.sinceLastSpecialKey += 1
		self.content = ebiten.AppendInputChars(self.content)
	}

	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 2, 1, 0, 255 })

	// draw text's area rectangle
	self.text.SetSize(18) // important for measuring!
	bounds := screen.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	x, y := h/32, h/32
	rect := self.text.Measure(string(self.content))
	rect  = rect.AddInts(x, y)
	area := screen.SubImage(rect.ImageRect()).(*ebiten.Image)
	area.Fill(color.RGBA{ 8, 72, 88, 255 })

	// draw text
	self.text.SetAlign(etxt.Top | etxt.Left)
	self.text.Draw(screen, string(self.content), x, y)

	// draw fps and other info for fun
	self.text.SetSize(14)
	self.text.SetAlign(etxt.Right | etxt.Baseline)
	info := fmt.Sprintf("%d glyphs - %.2fFPS", len(self.content), ebiten.ActualFPS())
	pad := int((ebiten.DeviceScaleFactor()*float64(h))/64)
	self.text.Draw(screen, info, w - pad, h - pad)
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
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white
	renderer.SetFont(sfntFont)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/measure")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game{
		text: renderer, 
		content: []rune("Interactive text"),
	})
	if err != nil { log.Fatal(err) }
}
