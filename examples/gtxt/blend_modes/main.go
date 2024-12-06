//go:build gtxt

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
)

// Must be compiled with '-tags gtxt'

const Alpha = 255 // can be changed (e.g. 144) if you want to see how
// color modes work with semi-transparency too

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetSize(24)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)

	// create target image and fill it with different colors
	target := image.NewRGBA(image.Rect(0, 0, 720, 300))
	for i := 0; i < 720*100*4; i += 4 { // first 100 lines cyan
		//target.Pix[i + 0] = 0
		target.Pix[i+1] = 255
		target.Pix[i+2] = 255
		target.Pix[i+3] = 255
	}
	for i := 720 * 100 * 4; i < 720*200*4; i += 4 { // next 100 lines magenta
		target.Pix[i+0] = 255
		//target.Pix[i + 1] = 0
		target.Pix[i+2] = 255
		target.Pix[i+3] = 255
	}
	for i := 720 * 200 * 4; i < 720*300*4; i += 4 { // next 100 lines yellow
		target.Pix[i+0] = 255
		target.Pix[i+1] = 255
		//target.Pix[i + 2] = 0
		target.Pix[i+3] = 255
	}

	// draw first row of blend modes
	offX, offY := 180, 100
	x := offX / 2
	y := offY / 2
	renderer.SetColor(color.RGBA{0, 0, 0, Alpha})
	renderer.Draw(target, "over", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendCut)
	renderer.Draw(target, "cut", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendReplace)
	renderer.Draw(target, "replace", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendHue)
	renderer.Draw(target, "hue", x, y)

	// draw second row of blend modes
	y += offY
	x = offX / 2
	renderer.SetColor(color.RGBA{0, Alpha, Alpha, Alpha})
	renderer.SetBlendMode(etxt.BlendSub)
	renderer.Draw(target, "subtract", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendAdd)
	renderer.Draw(target, "add", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendMultiply)
	renderer.Draw(target, "multiply", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendOver)
	renderer.Draw(target, "over", x, y)

	// draw third row of blend modes
	y += offY
	x = offX / 2
	renderer.SetColor(color.RGBA{Alpha, 0, 0, Alpha})
	renderer.SetBlendMode(etxt.BlendOver)
	renderer.Draw(target, "over", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendHue)
	renderer.Draw(target, "hue", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendSub)
	renderer.Draw(target, "subtract", x, y)

	x += offX
	renderer.SetBlendMode(etxt.BlendMultiply)
	renderer.Draw(target, "multiply", x, y)

	// store image as png
	filename, err := filepath.Abs("gtxt_blend_modes.png")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = png.Encode(file, target)
	if err != nil {
		log.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("Program exited successfully.\n")
}
