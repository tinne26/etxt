package main

import "image/color"
import "time"
import "log"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"

// NOTICE: this is the example from the readme, but it's not
//         easy to use, as it works differently from others and
//         expects specific fonts and paths. You probably don't
//         want to go changing all this manually.

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
