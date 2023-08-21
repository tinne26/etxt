package main

import "os"
import "log"
import "fmt"
import "math"
import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"

// This example showcases how to use the Twine type instead of
// a string in order to apply different formats and styles in
// a single draw call.
//
// The code in this example is very basic and only shows to use the 
// most basic twine commands. Much more advanced functions are both
// pre-implemented in etxt and allowed to be implemented by etxt
// users. See ebiten/twine_all. Another example, bbcode, also shows
// how to create custom types and functions to make twines easier to
// build when we want to tailor them to some specific usage context.
// 
// You can run this example like this:
//   go run github.com/tinne26/etxt/examples/ebiten/text@latest path/to/regular-font.ttf path/to/bold-font.ttf

type Game struct {
	text *etxt.Renderer
	twine etxt.Twine
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error { return nil }

func (self *Game) Draw(canvas *ebiten.Image) {
	canvas.Fill(color.RGBA{ 245, 253, 198, 255 })
	self.text.Complex().Draw(canvas, self.twine, 16, 32)
}

func main() {
	// get font path
	if len(os.Args) != 3 {
		msg := "Usage: expects two arguments with the paths to the two fonts to be used (regular and bold)\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse fonts
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil { log.Fatal(err) }
	sub, err := font.GetSubfamily(sfntFont)
	if err != nil { log.Fatal(err) }
	if sub != "Regular" {
		log.Fatalf("Expected first font to be 'Regular', but got '%s' instead.", sub)
	}
	boldFont, boldName, err := font.ParseFromPath(os.Args[2])
	if err != nil { log.Fatal(err) }
	sub, err = font.GetSubfamily(boldFont)
	if err != nil { log.Fatal(err) }
	if sub != "Bold" {
		log.Fatalf("Expected second font to be 'Bold', but got '%s' instead.", sub)
	}
	fmt.Printf("Fonts loaded: %s, %s\n", fontName, boldName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetColor(color.RGBA{7, 9, 15, 255})
	renderer.SetFont(sfntFont)
	renderer.SetSize(16)
	renderer.SetAlign(etxt.Baseline | etxt.Left)
	
	// configure additional bold font
	bold := renderer.Complex().RegisterFont(etxt.NextFontIndex, boldFont)

	// prepare some variables for the twine content
	var glyphs []sfnt.GlyphIndex
	for i := 0; i < 16; i++ { glyphs = append(glyphs, sfnt.GlyphIndex(i)) }
	caramel := color.RGBA{191, 148, 74, 255}

	// --- create twine content ---
	// This is the key part of this example. We are only using basic methods,
	// but we are making quite a mess of it in order to show all the main
	// possibilities. In practice you will be more normal and consistent.
	
	twine := etxt.Weave(
		bold, "Font name: ", etxt.Pop, fontName, '\n',
		bold, "Glyphs 0 - 15:", etxt.Pop, " '", glyphs, "'\n\n",
		bold, "Sample text:", etxt.Pop, '\n',
		"Twines are like strings, but they allow you to add ", caramel,
		"formatting\ndirectives", etxt.Pop, " on the text, encode glyphs ",
		"directly for text shaping,\npass strings as byte ",
		[]byte{'s', 'l', 'i', 'c', 'e', 's'}, ", create custom text ",
		"effects and more.",
		etxt.PopAll, // this is redundant here, but added for illustrative purposes
	)
	twine.AddLineBreak()
	twine.AddLineBreak()
	twine.Add("There are many ways to encode the content:\n")
	twine.Weave("- Using Weave(), a very dynamic but somewhat unsafe function", '.')
	twine.AddRune('\n')
	twine.Add("- Chaining ").PushColor(caramel).Add("Twine API").Pop().Add(" methods.").AddLineBreak()
	twine.Buffer = append(twine.Buffer, []byte("- Going low level and touching what you ")...)
	twine.PushFont(bold).Add("shouldn't").Pop().AddRune('.')

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/twine")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game{
		text: renderer,
		twine: twine,
	})
	if err != nil { log.Fatal(err) }
}
