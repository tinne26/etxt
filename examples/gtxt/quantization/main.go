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
import "github.com/tinne26/etxt/font"

// Must be compiled with '-tags gtxt'

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetSize(20)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Top | etxt.Left)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create target image and fill it with white
	// (you could measure text and use its proper line height
	// to determine a more precise size for the output image)
	outImage := image.NewRGBA(image.Rect(0, 0, 640, 64))
	for i := 0; i < 640*64*4; i++ {
		outImage.Pix[i] = 255
	}

	// draw quantized text
	TextSample := "Horizontally quantized vs unquantized text."
	renderer.Draw(outImage, TextSample + " [quantized]", 8, 8)

	// disable horizontal quantization and draw again
	renderer.Fract().SetHorzQuantization(etxt.QtNone)
	renderer.Draw(outImage, TextSample + " [unquantized]", 8, 32)

	// store image as png
	filename, err := filepath.Abs("gtxt_quantization.png")
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
