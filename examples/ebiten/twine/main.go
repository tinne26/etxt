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
import "github.com/tinne26/etxt/fract"

// This example showcases how to use the Twine type instead of
// a string in order to apply different formats and styles in
// a single draw call.
//
// The code in this example is very basic and only shows how to use
// the most basic twine commands. Much more advanced functions are
// both pre-implemented in etxt and allowed to be implemented through 
// the API. See ebiten/twine_demo. Another example, ebiten/bbcode,
// shows how to create custom types and functions to make twines easier 
// to build when we want to tailor them to some specific usage context.
// 
// You can run the example like this (you need two fonts):
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
	self.text.Twine().Draw(canvas, self.twine, 16, 32)
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
	
	// register fonts and functions that we will be using
	bold := renderer.Twine().RegisterFont(etxt.NextFontIndex, boldFont)
	spoiler := renderer.Twine().RegisterEffectFunc(etxt.NextEffectKey, (&Spoiler{}).Func)

	// prepare some variables for the twine content
	var glyphs []sfnt.GlyphIndex
	for i := 0; i < 16; i++ { glyphs = append(glyphs, sfnt.GlyphIndex(i)) }
	caramel := color.RGBA{166, 127, 58, 255} // very dark caramel actually
	// tea := color.RGBA{221, 232, 185, 255}
	// carmine := color.RGBA{147, 22, 33, 255}

	// --- create twine content ---
	// This is the key part of this example. We are only using basic methods,
	// but we are making quite a mess of it in order to show all the main
	// possibilities. In practice you will be more normal and consistent.
	
	twine := etxt.Weave(
		bold, "Font name: ", etxt.Pop, fontName, '\n',
		bold, "Glyphs 0 - 15:", etxt.Pop, " '", glyphs, "'\n\n",
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
	twine.Weave("- Using Weave(), a very dynamic but slightly unsafe function", '.')
	twine.AddRune('\n')
	twine.Add("- Chaining ").PushColor(caramel).Add("Twine API").Pop().Add(" methods one after another.")
	twine.AddLineBreak()
	twine.Add("- Using ").PushColor(caramel).Add("type embedding").Pop().Add(" and custom twine builders.")
	twine.AddLineBreak()
	twine.Buffer = append(twine.Buffer, []byte("- Going low level and touching what you ")...) // *
	// * This is generally not safe, you would have to know about internal implementation details.
	//   For example, if we had been adding glyphs right before the append this wouldn't work.
	twine.PushFont(bold).Add("shouldn't").Pop().AddRune('.')

	twine.Add("\n\n")
	twine.Add("Manipulating size can be ").ShiftSize(3).Add("dangerous").Pop().Add(", but it's possible with\n")
	twine.ShiftSize(-3).Add("certain restrictions").Pop().Add(". In particular, line height remains fixed to\n")
	twine.Add("the initial value unless you explicitly refresh it.\n\n")
	twine.Add("Not to spoil your life movie, but the main character ")
	twine.PushEffect(spoiler, etxt.SinglePass).Add("trips\nover twines and sustains a traumatic brain injury")
	twine.Pop().Add(".")

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/twine")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game{
		text: renderer,
		twine: twine,
	})
	if err != nil { log.Fatal(err) }
}

// A simple example of a type that implements a single-use TwineEffectFunc.
// Notice that this is implemented in a very primitive way. There's no
// support for multiple uses in a single twine, no reset functionality, etc.
// 
// If you want more effect examples, see etxt/twine_builtin_effects.go.
type Spoiler struct { uncovered, nextUncovered bool }
func (self *Spoiler) Func(renderer *etxt.Renderer, target etxt.Target, args etxt.TwineEffectArgs) {
	// prevent misuse
	args.AssertMode(etxt.SinglePass)
	
	// bypass if measuring
	if args.Measuring() { return }

	// handle each trigger situation
	switch args.GetTrigger() {
	case etxt.TwineTriggerPush:
		self.uncovered = self.nextUncovered
		self.nextUncovered = false
	case etxt.TwineTriggerPop, etxt.TwineTriggerLineBreak:
		rect := args.Rect()
		if !self.uncovered {
			rect.Clip(target).Fill(color.RGBA{20, 20, 20, 255})
		}
		x, y := ebiten.CursorPosition()
		inRect := rect.Contains(fract.IntsToPoint(x, y))
		self.nextUncovered = self.nextUncovered || inRect
	case etxt.TwineTriggerLineStart:
		// ignored
	default:
		panic("unexpected")
	}
}
