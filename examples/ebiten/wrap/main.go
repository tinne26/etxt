package main

import "os"
import "log"
import "fmt"
import "image"
import "unicode/utf8"

import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt"
import "github.com/hajimehoshi/ebiten/v2"

// CTRL + F "**" to jump to the explanation of the example.

// type alias to create an unexported alias of etxt.Renderer
type renderer = etxt.Renderer

// wrapper type for etxt.Renderer, for which we will add a DrawInBox() method
type TextBoxRenderer struct { renderer }

// does basic line wrapping
func (self *TextBoxRenderer) DrawInBox(text string, bounds image.Rectangle) {
	// helper function
	var getNextWord = func(str string, index int) string {
		start := index
		for index < len(str) {
			codePoint, size := utf8.DecodeRuneInString(str[index : ])
			if codePoint <= ' ' { return str[start : index] }
			index += size
		}
		return str[start : index]
	}

	// create Feed and iterate each rune / word
	feed := self.renderer.NewFeed(fixed.P(bounds.Min.X, bounds.Min.Y))
	index := 0
	for index < len(text) {
		// handle space case with Advance() instead of Draw()
		switch text[index] {
		case ' ':
			feed.Advance(' ')
			index += 1
		case '\n', '\r':
			feed.LineBreak()
			index += 1
		default:
			// get next word and measure it to see if it fits
			word := getNextWord(text, index)
			width := self.renderer.SelectionRect(word).Width
			if (feed.Position.X + width).Ceil() > bounds.Max.X {
				feed.LineBreak() // didn't fit, jump to next line before drawing
			}

			// abort if we are going beyond the vertical working area
			if feed.Position.Y.Floor() >= bounds.Max.Y { return }

			// draw the word and increase index
			for _, codePoint := range word {
				feed.Draw(codePoint)
			}
			index += len(word)
		}
	}
}

// ---- game and main code ----

type Game struct {
	txtRenderer *TextBoxRenderer
}

func (self *Game) Layout(w int, h int) (int, int) { return w, h }
func (self *Game) Update() error { return nil }
func (self *Game) Draw(screen *ebiten.Image) {
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.DrawInBox( // **
		"This example performs text wrapping in order to draw text within a delimited area. Additionally, " +
		"it also shows how to embed the etxt.Renderer type in a custom struct that will still allow every " +
		"method of etxt.Renderer to be used directly, while also allowing us to define additional methods." + 
		"\n\nIn this case, we have added DrawInBox(text string, bounds image.Rectangle). Try to resize the " + 
		"screen and see how the text adapts to it. You may take this code as a reference and write your " + 
		"own text wrapping functions, as you often will have more specific needs." +
		"\n\nIn most cases, you will want to add some padding to the bounds to avoid text sticking to the " +
		"borders of your text area.",
		screen.Bounds(),
	)
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
	txtRenderer := &TextBoxRenderer{ *etxt.NewStdRenderer() }
	txtRenderer.SetCacheHandler(cache.NewHandler())
	txtRenderer.SetSizePx(16)
	txtRenderer.SetFont(font)
	txtRenderer.SetAlign(etxt.Top, etxt.Left)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/wrap")
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err = ebiten.RunGame(&Game{ txtRenderer })
	if err != nil { log.Fatal(err) }
}
