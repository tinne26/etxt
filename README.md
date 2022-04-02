# etxt
[![Go Reference](https://pkg.go.dev/badge/github.com/tinne26/etxt.svg)](https://pkg.go.dev/github.com/tinne26/etxt)

**NOTICE: Work in progress! In most ways the package is very mature and solid already, but there are still a few rough edges (most notably faux bold, EdgeMarker, and a couple important missing examples).**

**etxt** is a package for font management and text rendering in Golang designed to be used with the [**Ebiten**](https://github.com/hajimehoshi/ebiten) game engine.

While Ebiten already provides the [**ebiten/text**](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/text) package that makes *getting some text drawn on the screen* easy enough, **etxt** aims to help you actually understand what you are doing, doing it in a structured way, and giving you much more power and control.

Font rendering is a complex topic that often feels at odds with the design principles behind a language like Golang, but if you need to deal with it anyway and want to do it responsibly-*ish*, this package is a best effort to help you bridge the gap.

As a quick summary of what this package provides:
- Structured font management and usage through the `FontLibrary` and `Renderer` types... because having to create and manage new `font.Face`s just to change text size is *not* ok.
- Full control over glyph mask caching and rasterization (or just stay with the defaults!).
- A few custom rasterizers that allow you to draw faux-bold, oblique, ~~blurred and hollow text~~ (WIP). Not really "main features", though, only examples of what you can do with **etxt**.
- Lots of [examples](https://github.com/tinne26/etxt/tree/main/examples) and thorough documentation.


## Code example
Less talk and more code!
```Golang
package main

import "log"
import "time"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt"

type Game struct { txtRenderer *etxt.Renderer }
func (self *Game) Layout(int, int) (int, int) { return 400, 400 }
func (self *Game) Update() error { return nil }
func (self *Game) Draw(screen *ebiten.Image) {
	millis := time.Now().UnixMilli() // don't do this in actual games ;)
	blue := (millis/16)%512
	if blue >= 256 { blue = 511 - blue }
	changingColor := color.RGBA{ 0, 255, uint8(blue), 255 }

	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.SetColor(changingColor)
	self.txtRenderer.Draw("Hello World!", 200, 200)
}

func main() {
	// load font library
	fontLib := etxt.NewFontLibrary()
	_, _, err := fontLib.ParseDirFonts("game_dir/assets/fonts")
	if err != nil {
		log.Fatalf("Error while loading fonts: %s", err.Error())
	}

	// check that we have the fonts we want
	// (we are not using this many fonts in the example, though...)
	// (showing it for completeness, you don't need this in most cases)
	if !fontLib.HasFont("League Gothic Regular") { log.Fatal("missing font 1") }
	if !fontLib.HasFont("Carter One"           ) { log.Fatal("missing font 2") }
	if !fontLib.HasFont("Roboto Bold"          ) { log.Fatal("missing font 3") }

	// check that the fonts have the characters we want
	// (showing it for completeness, you don't need this in most cases)
	err = fontLib.EachFont(checkMissingRunes)
	if err != nil { log.Fatal(err) }

	// create a new text renderer and configure it
	txtRenderer    := etxt.NewStdRenderer()
	glyphsCache    := etxt.NewDefaultCache(10*1024*1024) // 10MB
	txtRenderer.SetCacheHandler(glyphsCache.NewHandler())
	txtRenderer.SetFont(fontLib.GetFont("League Gothic Regular"))
	txtRenderer.SetAlign(etxt.YCenter, etxt.XCenter)
	txtRenderer.SetSizePx(72)

	// run the "game"
	err = ebiten.RunGame(&Game{ txtRenderer })
	ebiten.SetWindowSize(400, 400)
	if err != nil {
		log.Fatalf("ebiten.RunGame error: %s", err.Error())
	}
}

// helper used after loading fonts
func checkMissingRunes(name string, font *etxt.Font) error {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 .,;:!?-()[]"

	missing, err := etxt.GetMissingRunes(font, alphabet)
	if err != nil { return err }
	if len(missing) > 0 {
		log.Fatalf("Font '%s' missing runes: %s", name, string(missing))
	}
	return nil
}
```

This example focuses on the mundane usage of the main **etxt** `FontLibrary` and `Renderer` types, with abundant checks to fail fast if anything seems out of place, but there are [many more examples](https://github.com/tinne26/etxt/tree/main/examples) (and much flashier) in the project, so check them out!


## Can I use this package without Ebiten?
Yeah, you can compile it with `-tags gtxt` (in fact, the Ebiten version could be a modified fork of the `gtxt` version instead, but that's a pain to manage... so I gave preference to the Ebiten version as my original target).

Notice that `gtxt` will make text drawing happen on the CPU, so don't try to use it for real-time stuff. In particular, be careful to not accidentally use `gtxt` with Ebiten (they are compatible in many cases, but performance will die).

## Should I bother learning to use etxt?
The difficult part is learning about fonts in general, not **etxt** in particular. If you are only dealing with text rendering incidentally and **ebiten/text** does the job well enough for you, I won't try to convince you to learn more about fonts, you probably have better things to spend your time on.

That said, if you want to know more and have some time to invest, here's my advice:
1. Spend an hour reading [FreeType glyph conventions](https://freetype.org/freetype2/docs/glyphs/index.html) up to section IV or V. Seriously, if you are interested in the topic but you don't read this you are just self-sabotaging.
2. Sleep on it.
3. Re-read 1.
4. Now you can go through this package's documentation and examples, and they shouldn't pose any problems.

## Any limitations I should be aware of?
- Colored glyphs like emojis are not supported. **sfnt** doesn't support them, but **etxt** is not designed to support them anyway (and in a game context, using images directly is perfectly appropriate as an alternative).
- No automatic support for bidirectional text. You can use [x/text/unicode/bidi](https://pkg.go.dev/golang.org/x/text/unicode/bidi) though, and then **etxt**'s `Renderer` allows you to set the rendering direction. See [examples/gtxt/direction_bidi](https://github.com/tinne26/etxt/blob/main/examples/gtxt/direction_bidi/main.go).
- **etxt** relies on [/x/image/font/sfnt](https://pkg.go.dev/golang.org/x/image/font/sfnt) under the hood, so it has the same limitations that **sfnt** has, which are significant. This will get technical, but here we go:
	- Hinting doesn't exist in **etxt** because what **sfnt** does isn't hinting yet. All **sfnt** is doing is quantizing glyph positions and measures, not trying to read TrueType hinting instructions or applying any techniques to improve the readability of glyphs when projected to the pixel grid.
	- Vertical text is not supported in any clean way because **sfnt** doesn't expose the relevant tables to determine vertical spacing between glyphs.
	- While **etxt** supports drawing text based on glyph indices (instead of only runes), there's a hole in Go's landscape when it comes to [text shaping](https://github.com/tinne26/etxt/blob/main/docs/shaping.md). **sfnt** doesn't expose enough information directly, so you might want to look into [go-text/typesetting](https://github.com/go-text/typesetting) instead.
	- You get the hang of it: https://github.com/golang/go/issues/45325.
- Glyph masks for Ebiten will be simplified (breaking compatibility) once Ebiten [accepts arbitrary bounds](https://github.com/hajimehoshi/ebiten/issues/2013) for its images.


## Any future plans?
If I ever get really bored, I'd like to look into:
- Contributing to Golang's **sfnt** to expose more tables and allow the creation of minimal packages to do basic text shaping in arabic or other complex scripts.
- Add outline expansion. Freetype and libASS do this, and it would be quite nice to get high quality outlines and better faux-bolds... but it's also *hard*; I don't really know if I want to go there.
- Triangulation and GPU rendering of Bézier curves are also interesting for Ebiten, although they probably don't belong here.
