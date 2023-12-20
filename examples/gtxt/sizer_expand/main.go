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
import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/sizer"

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
	renderer.SetSize(32)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white

	// create sizer and set it too
	var padSizer sizer.PaddedKernSizer
	renderer.SetSizer(&padSizer)

	// create target image and fill it with black
	outImage := image.NewRGBA(image.Rect(0, 0, 600, 230))
	for i := 3; i < 600*230*4; i += 4 { outImage.Pix[i] = 255 }

	// set target and draw each line expanding more and more
	for i := 0; i < 6; i++ {
		padSizer.SetPadding(fract.FromInt(i*12))
		renderer.Draw(outImage, "pyramid", 300, (i + 1)*32)

		// note: if we didn't have the sizer available in the scope,
		// we would simply have to retrieve it first:
		// >> sizer := renderer.GetSizer().(*sizer.PaddedKernSizer)
		// >> sizer.SetPadding(fract.FromInt(i*12))
	}

	// store image as png
	filename, err := filepath.Abs("gtxt_sizer_expand.png")
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
