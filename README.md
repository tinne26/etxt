# etxt
[![Go Reference](https://pkg.go.dev/badge/github.com/tinne26/etxt@v0.0.9-alpha.6.svg)](https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.6)

**NOTICE: this is a preview of v0.0.9, which is a non-trivial departure from previous versions. For the latest stable version, see [v0.0.8](https://github.com/tinne26/etxt/tree/v0.0.8).**

**etxt** is a package for [vectorial](https://github.com/tinne26/etxt/blob/main/docs/panorama.md)[^1] text rendering in Golang designed to be used with [**Ebitengine**](https://github.com/hajimehoshi/ebiten), the 2D game engine made by [Hajime Hoshi](https://github.com/hajimehoshi) for Golang.

[^1]: If you are using pixel-art-like vectorial fonts, read [these tips](https://github.com/tinne26/etxt/blob/main/docs/pixel-tips.md).

While Ebitengine already includes the [**ebiten/v2/text/v2**](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/text/v2) package, **etxt** offers some advantages over it:
- Makes text size easy to change.
- Puts emphasis on getting [display scaling](https://github.com/tinne26/etxt/blob/main/docs/display-scaling.md) right.
- Gets rid of `font.Face` for good.
- Provides high quality documentation and [examples](https://github.com/tinne26/etxt/tree/main/examples).
- Helps out with some extras like faux bold, faux oblique, basic line wrapping, embedded fonts, glyph quantization, line spacing, etc.
- Exposes caches, rasterizers and sizers for you to adapt if you have more advanced needs.

What **etxt** doesn't do:
- No general [text layout](https://raphlinus.github.io/text/2020/10/26/text-layout.html). Features like bidi, itemization, shaping, general hit testing, justification and others are not covered and in most cases aren't a primary goal for this package.
- Poor or no support for [complex scripts](https://github.com/tinne26/etxt/blob/main/docs/shaping.md) (e.g. Arabic) nor vertical text layouts (e.g. Japanese). Notice that [**ebiten/v2/text/v2**](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/text/v2) fares much better in this regard.
- None of the things people actually want: shadows and outlines, gamma correction, subpixel antialiasing, Knuth-Plass line breaking, better support for shaders, etc. Some can already be crudely faked, some will be added in the future... but this is the situation right now.

*If you are unfamiliar with typography terms and concepts, I highly recommend reading the first chapters of [FreeType Glyph Conventions](https://freetype.org/freetype2/docs/glyphs/index.html); one the best references on the topic you can find on the internet.*

## Code example
Less talk and more code!
```Golang
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
	
	// get screen center position
	bounds := canvas.Bounds() // assumes origin (0, 0)
	x, y := bounds.Dx()/2, bounds.Dy()/2

	// draw text
	word := Words[int(self.wordIndex)]
	self.text.Draw(canvas, word, x, y)
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
```

You can try running this yourself with[^2]:
```
go run github.com/tinne26/etxt/examples/ebiten/words@latest
```

[^2]: You will need Golang >=1.18, and if you have never used Ebitengine before, you may need to [install some dependencies](https://ebitengine.org/en/documents/install.html?os=linux) (typically only on Linux or FreeBSD).

This is a very simple and self-contained example. If you want to learn more, make sure to take a look at [etxt/examples](https://github.com/tinne26/etxt/tree/main/examples)!

## Can I use this package without Ebitengine?
Yeah, you can compile it with `-tags gtxt`. Notice that `gtxt` will make text drawing happen on the CPU, so don't try to use it for real-time applications. In particular, be careful to not accidentally use `gtxt` with Ebitengine (they are compatible in many cases, but performance will die).

## Testing, contributions and others
- For testing, see the instructions on [`etxt/test`](https://github.com/tinne26/etxt/blob/main/test).
- If you have any questions or suggestions for improvements feel free to speak, I'm always happy to explain or discuss.
- Otherwise, I'm not looking for contributors nor general help.
