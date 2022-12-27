package main

import "os"
import "log"
import "fmt"
import "math"
import "image"
import "unicode/utf8"

import "golang.org/x/image/math/fixed"
import "github.com/tinne26/etxt"
import "github.com/hajimehoshi/ebiten/v2"

// The explanation of the example is displayed in the example itself
// based on the contents of this string:
const Content = "This example performs basic text wrapping in order to draw text " +
                "within a delimited area. Additionally, it also shows how to embed " +
                "the etxt.Renderer type in a custom struct that allows defining our " + 
                "own methods while also preserving all the original methods of " +
                "etxt.Renderer.\n\nIn this case, we have added DrawInBox(text string, " +
                "bounds image.Rectangle). Try to resize the screen and see how the text " + 
                "adapts to it. You may take this code as a reference and write your own " +
                "text wrapping functions, as you often will have more specific needs." +
                "\n\nIn most cases, you will want to add some padding to the bounds to " +
                "avoid text sticking to the borders of your target text area."

// Type alias to create an unexported alias of etxt.Renderer.
// This is quite irrelevant for this example, but it's useful in
// practical scenarios to avoid leaking a public internal field.
type renderer = etxt.Renderer

// Wrapper type for etxt.Renderer. Since this type embeds etxt.Renderer
// it will preserve all its methods, and we can additionally add our own
// new DrawInBox() method.
type TextBoxRenderer struct { renderer }

// The new method for TextBoxRenderer. It draws the given text within the
// given bounds, performing basic line wrapping on space " " characters.
// This is only meant as a reference: this method doesn't split on "-",
// very long words will overflow the box when a single word is longer
// than the width of the box, \r\n will be considered two line breaks
// instead of one, etc. In many practical scenarios you will want to
// further customize the behavior of this function. For more complex
// examples of Feed usages, see examples/ebiten/typewriter, which also
// has a typewriter effect, multiple colors, bold, italics and more.
// Otherwise, if you only needed really basic line wrapping, feel free
// to copy this function and use it directly. If you don't want a custom
// TextBoxRenderer type, it's trivial to adapt the function to receive
// a standard *etxt.Renderer as an argument instead.
//
// Notice that this function relies on the renderer's alignment being
// (etxt.Top, etxt.Left).
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
		switch text[index] {
		case ' ': // handle spaces with Advance() instead of Draw()
			feed.Advance(' ')
			index += 1
		case '\n', '\r': // \r\n line breaks *not* handled as single line breaks
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
				feed.Draw(codePoint) // you may want to cut this earlier if the word is too long
			}
			index += len(word)
		}
	}
}

// ---- game and main code ----

type Game struct {
	txtRenderer *TextBoxRenderer
}

func (self *Game) Layout(w int, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	return int(math.Ceil(float64(w)*scale)), int(math.Ceil(float64(h)*scale))
}
func (self *Game) Update() error { return nil }
func (self *Game) Draw(screen *ebiten.Image) {
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.DrawInBox(Content, screen.Bounds())
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
	txtRenderer.SetSizePx(int(16*ebiten.DeviceScaleFactor()))
	txtRenderer.SetFont(font)
	txtRenderer.SetAlign(etxt.Top, etxt.Left) // important for this example!

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/wrap")
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err = ebiten.RunGame(&Game{ txtRenderer })
	if err != nil { log.Fatal(err) }
}
