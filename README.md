# etxt
[![Go Reference](https://pkg.go.dev/badge/github.com/tinne26/etxt.svg)](https://pkg.go.dev/github.com/tinne26/etxt)

**etxt** is a package for font management and text rendering in Golang designed to be used with the [**Ebitengine**](https://github.com/hajimehoshi/ebiten) game engine.

While Ebitengine already provides the [**ebiten/text**](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/text) package that makes *getting some text drawn on screen* easy enough, **etxt** aims to help you actually understand what you are doing, doing it in a structured way and giving you much more control over it.

As a quick summary of what this package provides:
- Structured font management and usage through the `FontLibrary` and `Renderer` types; because having to create and manage new `font.Face`s just to change text size is *not* ok.
- Full control over glyph mask caching and rasterization (or just stay with the defaults!).
- A few custom rasterizers that allow you to draw faux-bold, oblique, ~~blurred and hollow text~~ (WIP). Not really "main features", though, only examples of what you can do with **etxt**.
- Lots of [examples](https://github.com/tinne26/etxt/tree/main/examples) and thorough documentation.

## Code example
Less talk and more code!
```Golang
package main

import ( "log" ; "time" ; "image/color" )
import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt"

type Game struct { txtRenderer *etxt.Renderer }
func (self *Game) Layout(int, int) (int, int) { return 400, 400 }
func (self *Game) Update() error { return nil }
func (self *Game) Draw(screen *ebiten.Image) {
	// hacky color computation
	millis := time.Now().UnixMilli()
	blue := (millis/16) % 512
	if blue >= 256 { blue = 511 - blue }
	changingColor := color.RGBA{ 0, 255, uint8(blue), 255 }

	// set relevant text renderer properties and draw
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.SetColor(changingColor)
	self.txtRenderer.Draw("Hello World!", 200, 200)
}

func main() {
	// load font library
	fontLib := etxt.NewFontLibrary()
	_, _, err := fontLib.ParseDirFonts("game_dir/assets/fonts") // !!
	if err != nil {
		log.Fatalf("Error while loading fonts: %s", err.Error())
	}

	// check that we have the fonts we want
	// (shown for completeness, you don't need this in most cases)
	expectedFonts := []string{ "Roboto Bold", "Carter One" }  // !!
	for _, fontName := range expectedFonts {
		if !fontLib.HasFont(fontName) {
			log.Fatal("missing font: " + fontName)
		}
	}

	// check that the fonts have the characters we want
	// (shown for completeness, you don't need this in most cases)
	err = fontLib.EachFont(checkMissingRunes)
	if err != nil { log.Fatal(err) }

	// create a new text renderer and configure it
	txtRenderer := etxt.NewStdRenderer()
	glyphsCache := etxt.NewDefaultCache(10*1024*1024) // 10MB
	txtRenderer.SetCacheHandler(glyphsCache.NewHandler())
	txtRenderer.SetFont(fontLib.GetFont(expectedFonts[0]))
	txtRenderer.SetAlign(etxt.YCenter, etxt.XCenter)
	txtRenderer.SetSizePx(64)

	// run the "game"
	ebiten.SetWindowSize(400, 400)
	err = ebiten.RunGame(&Game{ txtRenderer })
	if err != nil { log.Fatal(err) }
}

// helper function used with FontLibrary.EachFont to make sure
// all loaded fonts contain the characters or alphabet we want
func checkMissingRunes(name string, font *etxt.Font) error {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const symbols = "0123456789 .,;:!?-()[]{}_&#@"

	missing, err := etxt.GetMissingRunes(font, letters + symbols)
	if err != nil { return err }
	if len(missing) > 0 {
		log.Fatalf("Font '%s' missing runes: %s", name, string(missing))
	}
	return nil
}
```

This example focuses on the mundane usage of the main **etxt** `FontLibrary` and `Renderer` types, with abundant checks to fail fast if anything seems out of place.

If you want flashier examples you will find [many more](https://github.com/tinne26/etxt/tree/main/examples) in the project, make sure to check them out!

## Can I use this package without Ebitengine?
Yeah, you can compile it with `-tags gtxt`. Notice that `gtxt` will make text drawing happen on the CPU, so don't try to use it for real-time stuff. In particular, be careful to not accidentally use `gtxt` with Ebitengine (they are compatible in many cases, but performance will die).

## Should I bother learning to use etxt?
If you are only dealing with text rendering incidentally and **ebiten/text** does the job well enough for you, feel free to stay with that.

The main consideration when using **etxt** is that you need to be minimally acquainted with how fonts work. [FreeType glyph conventions](https://freetype.org/freetype2/docs/glyphs/index.html) is the go to reference that you *really should be reading* (up to section IV or V).

## Any future plans?
This package is already quite solid, there are only a few points left to improve:
- Adding a few more effects (hollow text, shaders, etc).
- Missing a couple important examples (crisp UI and shaders).

If I get really bored, I'd also like to look into:
- Contributing to Golang's **sfnt** to [expose more tables](https://github.com/golang/go/issues/45325) and allow the creation of minimal packages to do basic [text shaping](https://github.com/tinne26/etxt/blob/main/docs/shaping.md) in arabic or other complex scripts.
- Add outline expansion. Freetype and libASS do this, and it would be quite nice to get high quality outlines and better faux-bolds... but it's also *hard*; I don't really know if I want to go there.
- Triangulation and GPU rendering of Bézier curves are also interesting for Ebitengine (although they probably don't belong in this package).

## Testing, contributions and others
- For testing, see the instructions on [`etxt/test`](https://github.com/tinne26/etxt/blob/main/test).
- If you have any questions or suggestions for improvements feel free to ask, I'm always happy to explain or discuss.
- I'm not looking for contributors nor general help.
- The API is reasonably stable, but I'll never hesitate to break compatibility if it's to make the library better. I also tend to update dependency versions when tagging new versions.
