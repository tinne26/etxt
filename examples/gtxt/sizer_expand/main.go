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
import "github.com/tinne26/etxt/esizer"

// Must be compiled with '-tags gtxt'

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
	renderer.SetSizePx(32)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white

	// create sizer and set it too
	padSizer := &esizer.HorzPaddingSizer{}
	renderer.SetSizer(padSizer)

	// create target image and fill it with black
	outImage := image.NewRGBA(image.Rect(0, 0, 600, 230))
	for i := 3; i < 600*230*4; i += 4 { outImage.Pix[i] = 255 }

	// set target and draw each line expanding more and more
	renderer.SetTarget(outImage)
	for i := 0; i < 6; i++ {
		padSizer.SetHorzPadding(i*12) // *
		renderer.Draw("pyramid", 300, (i + 1)*32)

		// * alternative code if we didn't have the sizer locally:
		// sizer := renderer.GetSizer().(*esizer.HorzPaddingSizer)
		// sizer.SetHorzPadding(i*12)
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
