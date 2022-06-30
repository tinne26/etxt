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
import "github.com/tinne26/etxt/emask"

// Must be compiled with '-tags gtxt'

// NOTICE: the OutlineRasterizer is still experimental and it doesn't
//         handle all the edge cases properly yet. This example should
//         only be considered a preview.

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
	outliner := emask.NewOutlineRasterizer(1.0)
	renderer := etxt.NewRenderer(outliner)
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(72)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// create target image and fill it with white
	outImage := image.NewRGBA(image.Rect(0, 0, 512, 96))
	for i := 0; i < 512*96*4; i++ { outImage.Pix[i] = 255 }

	// set target and draw
	renderer.SetTarget(outImage)
	renderer.SetColor(color.RGBA{255, 0, 0, 255})
	renderer.Draw("Nice Outline!", 256, 48)

	// store result as png
	filename, err := filepath.Abs("gtxt_outline.png")
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
