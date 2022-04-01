//go:build gtxt

package main

import "os"
import "image"
import "image/color"
import "image/png"
import "path/filepath"
import "log"
import "fmt"

import "github.com/tinne26/etxt"

// Must be compiled with '-tags gtxt'

// TODO: it would be nice to make an extra example with semi-transparent
//       colors, mostly to confirm that all the modes are working as
//       expected, which I don't trust very much.

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
	renderer := etxt.NewStdRenderer()
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(24)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create target image and fill it with different colors
	outImage := image.NewRGBA(image.Rect(0, 0, 720, 300))
	for i := 0; i < 720*100*4; i += 4 { // first 100 lines cyan
		//outImage.Pix[i + 0] = 0
		outImage.Pix[i + 1] = 255
		outImage.Pix[i + 2] = 255
		outImage.Pix[i + 3] = 255
	}
	for i := 720*100*4; i < 720*200*4; i += 4 { // next 100 lines magenta
		outImage.Pix[i + 0] = 255
		//outImage.Pix[i + 1] = 0
		outImage.Pix[i + 2] = 255
		outImage.Pix[i + 3] = 255
	}
	for i := 720*200*4; i < 720*300*4; i += 4 { // next 100 lines yellow
		outImage.Pix[i + 0] = 255
		outImage.Pix[i + 1] = 255
		//outImage.Pix[i + 2] = 0
		outImage.Pix[i + 3] = 255
	}

	// set target
	renderer.SetTarget(outImage)

	// draw first row of mix modes
	offX, offY := 180, 100
	x := offX/2 ; y := offY/2
	renderer.Draw("over", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixCut)
	renderer.Draw("cut", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixReplace)
	renderer.Draw("replace", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixFiftyFifty)
	renderer.Draw("50%-50%", x, y)

	// draw second row of mix modes
	y += offY
	x  = offX/2
	renderer.SetColor(color.RGBA{0, 255, 255, 255})
	renderer.SetMixMode(etxt.MixSub)
	renderer.Draw("subtract", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixAdd)
	renderer.Draw("add", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixMultiply)
	renderer.Draw("multiply", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixOver)
	renderer.Draw("over", x, y)

	// draw third row of mix modes
	y += offY
	x  = offX/2
	renderer.SetColor(color.RGBA{255, 0, 0, 255})
	renderer.SetMixMode(etxt.MixOver)
	renderer.Draw("over", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixFiftyFifty)
	renderer.Draw("50%-50%", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixSub)
	renderer.Draw("subtract", x, y)

	x += offX
	renderer.SetMixMode(etxt.MixMultiply)
	renderer.Draw("multiply", x, y)

	// store image as png
	filename, err := filepath.Abs("gtxt_mix_modes.png")
	if err != nil { log.Fatal(err) }
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil { log.Fatal(err) }
	err = png.Encode(file, outImage)
	if err != nil { log.Fatal(err) }
	err = file.Close()
	if err != nil { log.Fatal(err) }
	fmt.Print("Program exited successfully.\n")
}
