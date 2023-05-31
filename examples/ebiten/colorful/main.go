package main

import "os"
import "log"
import "fmt"
import "math"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/font"

// This example draws some text with a color changing effect,
// where each letter changes color progressively. You can
// run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/colorful@latest path/to/font.ttf
//
// This example showcases how to use RendererGlyph.SetDrawFunc(),
// creating a custom drawing function to manually alter the color
// of each letter in the text. For other examples of SetDrawFunc(),
// see also examples/ebiten/shaking.
//
// Notice that changing text color through SetDrawFunc() is rather
// unusual, and in most cases you will prefer using a Feed or creating
// a complex Text object with color changing indications. In this
// case, though, since we want to change the color *of each letter*
// in a dynamic and continuous way, customizing the glyph drawing
// function directly feels more natural.

type Game struct {
	text *etxt.Renderer
	
	// text color variables
	red   float64
	green float64
	blue  float64
	shift float64
}

func (self *Game) Layout(winWidth int, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	// progressively change the values used in Draw to
	// determine letter colors, using different speeds
	self.red   -= 0.0202
	self.green -= 0.0168
	self.blue  -= 0.0227
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw text
	bounds := screen.Bounds()
	self.shift = 0.0 // reset color shift factor
	self.text.Draw(screen, "Colorful!\nWonderful!", bounds.Dx()/2, bounds.Dy()/2)
}

// This is the function that we use to override the text renderer's default draw
// function. It's set on the main through renderer.Glyph().SetDrawFunc().
func (self *Game) GlyphDrawFunc(target etxt.TargetImage, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
	// derive the color for the current letter from the initial/ values on
	// each color channel, the current offset, and the sine function
	r := (math.Sin(self.red + self.shift) + 1.0)/2.0
	g := (math.Sin(self.green + self.shift) + 1.0)/2.0
	b := (math.Sin(self.blue + self.shift) + 1.0)/2.0
	textColor := color.RGBA{uint8(r*255), uint8(g*255), uint8(b*255), 255}
	self.text.SetColor(textColor) // *
	// * Not all renderer properties are safe to change during drawing,
	//   but color is one of the exceptions.

	// draw the glyph mask
	mask := self.text.Glyph().LoadMask(glyphIndex, origin)
	self.text.Glyph().DrawMask(target, mask, origin)

	// increase offset to apply to the next letters
	self.shift += 0.15
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

	// create game struct
	game := &Game{
		text: renderer,
		red: -5.54,
		green: -4.3,
		blue: -6.4,
	}

	// override default text renderer draw function
	renderer.Glyph().SetDrawFunc(game.GlyphDrawFunc)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/colorful")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(game)
	if err != nil { log.Fatal(err) }
}
