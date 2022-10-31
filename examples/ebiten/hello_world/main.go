package main

import ( "log" ; "time" ; "image/color" )
import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt"

// NOTICE: this is the example from the readme, but it's not
//         easy to use, as it works differently from others and
//         expects specific fonts and paths. You probably don't
//         want to go changing all this manually... but if you
//         do, check the lines with !! comments at the end.

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
		log.Fatal("Error while loading fonts: " + err.Error())
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
