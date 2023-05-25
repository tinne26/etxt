package main

import ( "math" ; "image/color" )
import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt"
import "github.com/tinne26/fonts/liberation/lbrtserif"

const WordsPerSec = 2.71828
var Words = []string {
	"solitude", "joy", "ride", "whisper", "leaves", "cookie",
	"hearts", "disdain", "simple", "death", "sea", "shallow",
	"self", "rhyme", "childish", "sky", "tic", "tac", "boom",
}

// ---- Ebitengine's Game interface implementation ----

type Game struct { text *etxt.Renderer ; wordIndex float64 }

func (self *Game) Layout(winWidth int, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	newIndex := (self.wordIndex + WordsPerSec/60.0)
	self.wordIndex = math.Mod(newIndex, float64(len(Words)))
	return nil
}

func (self *Game) Draw(canvas *ebiten.Image) {
	// background color
	canvas.Fill(color.RGBA{229, 255, 222, 255})
	
	// get screen center position and text content
	bounds := canvas.Bounds() // assumes origin (0, 0)
	x, y := bounds.Dx()/2, bounds.Dy()/2
	text := Words[int(self.wordIndex)]

	// draw the text
	self.text.Draw(canvas, text, x, y)
}

// ---- main function ----

func main() {
	// create text renderer, set the font and cache
	renderer := etxt.NewRenderer()
	renderer.SetFont(lbrtserif.Font())
	renderer.Utils().SetCache8MiB()
	
	// adjust main text style properties
	renderer.SetColor(color.RGBA{239, 91, 91, 255})
	renderer.SetAlign(etxt.Center)
	renderer.SetSize(72)

	// set up Ebitengine and start the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/words")
	err := ebiten.RunGame(&Game{ text: renderer })
	if err != nil { panic(err) }
}
