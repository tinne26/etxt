package main

import "os"
import "log"
import "fmt"
import "math"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/inpututil"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"

// This example allows you to interactively write text in an Ebitengine
// program and see how the measurement rect for the text changes. You
// can use backspace to remove characters and enter to create line
// breaks. You can run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/measure@latest path/to/font.ttf

type Game struct {
	text *etxt.Renderer
	content []rune // not very efficient, but AppendInputChars uses runes
	wrapMode bool
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	var keyRepeat = func(key ebiten.Key) bool {
		ticks := inpututil.KeyPressDuration(key)
		return ticks == 1 || (ticks > 14 && (ticks - 14) % 9 == 0)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyControlLeft) || inpututil.IsKeyJustPressed(ebiten.KeyControlRight) {
		self.wrapMode = !self.wrapMode
	}

	if keyRepeat(ebiten.KeyBackspace) && len(self.content) >= 1 {
		self.content = self.content[0 : len(self.content) - 1]
	} else if keyRepeat(ebiten.KeyEnter) {
		self.content = append(self.content, '\n')
	} else {
		self.content = ebiten.AppendInputChars(self.content)
	}

	return nil
}

func (self *Game) Draw(canvas *ebiten.Image) {
	// dark background
	canvas.Fill(color.RGBA{ 2, 1, 0, 255 })

	// get canvas size and basic coords
	bounds := canvas.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	pad := int((ebiten.DeviceScaleFactor()*float64(h))/64)
	x, y := pad*2, pad*2
	
	//  highlight text's area rectangle and draw text
	areaColor := color.RGBA{ 8, 72, 88, 255 }
	content := string(self.content)
	if self.wrapMode { // measure and draw
		maxLineWidth := w - 2*x
		textArea := self.text.MeasureWithWrap(content, maxLineWidth)
		textArea.AddInts(x, y).Clip(canvas).Fill(areaColor)
		self.text.DrawWithWrap(canvas, content, x, y, maxLineWidth)
	} else {
		textArea := self.text.Measure(content)
		textArea.AddInts(x, y).Clip(canvas).Fill(areaColor)
		self.text.Draw(canvas, content, x, y)
	}

	// draw instructions, fps and other info for fun
	self.text.Utils().StoreState()
	defer self.text.Utils().RestoreState()
	self.text.SetSize(14)
	self.text.SetAlign(etxt.Right | etxt.Baseline)
	var info string
	fps := ebiten.ActualFPS()
	if self.wrapMode {
		info = fmt.Sprintf("%d glyphs - %.2fFPS | Line Wrap On [CTRL]", len(self.content), fps)
	} else {
		info = fmt.Sprintf("%d glyphs - %.2fFPS | Line Wrap Off [CTRL]", len(self.content), fps)
	}
	self.text.Draw(canvas, info, w - pad, h - pad)
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
	renderer.SetSize(18)
	renderer.SetAlign(etxt.Top | etxt.Left)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/measure")
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err = ebiten.RunGame(&Game{
		text: renderer, 
		content: []rune("Interactive text"),
	})
	if err != nil { log.Fatal(err) }
}
