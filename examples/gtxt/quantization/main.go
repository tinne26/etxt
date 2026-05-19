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

const TextSample = "Horizontally quantized vs unquantized text."

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
	renderer.SetSize(20)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.VertCenter | etxt.Left)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create target image and fill it with white
	lineHeight := renderer.Metrics().LineHeight().ToIntCeil()
	w := renderer.Measure(TextSample+" [unquantized]").IntWidth() + 24
	h := lineHeight*2 + lineHeight/4
	outImage := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h*4; i++ {
		outImage.Pix[i] = 255
	}

	// draw quantized text
	renderer.Fract().SetHorzQuantization(etxt.QtFull)
	renderer.Draw(outImage, TextSample+" [quantized]", 8, h/3)

	// disable horizontal quantization and draw again
	renderer.Fract().SetHorzQuantization(etxt.QtNone)
	renderer.Draw(outImage, TextSample+" [unquantized]", 8, h-h/3)

	// store image as png
	filename, err := filepath.Abs("gtxt_quantization.png")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = png.Encode(file, outImage)
	if err != nil {
		log.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("Program exited successfully.\n")
}
