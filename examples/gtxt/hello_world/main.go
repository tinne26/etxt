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

func main() {
	const OutImgWidth  = 256
	const OutImgHeight = 64
	const TextSizePx   = 32

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
	renderer.SetSizePx(TextSizePx)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create target image and fill it with white
	outImage := image.NewRGBA(image.Rect(0, 0, OutImgWidth, OutImgHeight))
	for i := 0; i < OutImgWidth*OutImgHeight*4; i++ {
		outImage.Pix[i] = 255
	}

	// set target and draw
	renderer.SetTarget(outImage)
	renderer.Draw("Hello World!", OutImgWidth/2, OutImgHeight/2)

	// store image as png
	filename, err := filepath.Abs("gtxt_hello_world.png")
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
