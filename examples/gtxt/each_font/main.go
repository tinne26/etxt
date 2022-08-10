package main

import "os"
import "sort"
import "image"
import "image/color"
import "image/png"
import "path/filepath"
import "log"
import "fmt"

import "github.com/tinne26/etxt"

// Must be compiled with '-tags gtxt'.
// This example expects a path to a font directory as the first
// argument, reads the fonts in it and creates an image where each
// font name is drawn with its own font.

func main() {
	// get font directory path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font directory\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// print given font directory
	fontDir, err := filepath.Abs(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Reading font directory: %s\n", fontDir)

	// create font library, parsing fonts in the given directory
	fontLib := etxt.NewFontLibrary()
	added, skipped, err := fontLib.ParseDirFonts(fontDir)
	if err != nil {
		log.Fatalf("Added %d fonts, skipped %d, failed with '%s'", added, skipped, err.Error())
	}

	// create renderer (uncached in this example)
	renderer := etxt.NewStdRenderer()
	renderer.SetSizePx(24)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// determine how much space we will need to draw all
	// the fonts while also collecting their names
	width, height := 0, 0
	names := make([]string, 0, fontLib.Size())
	err = fontLib.EachFont(
		func(fontName string, font *etxt.Font) error {
			renderer.SetFont(font)
			rect := renderer.SelectionRect(fontName)
			height += rect.Height.Ceil()
			if rect.Width.Ceil() > width { width = rect.Width.Ceil() }
			names = append(names, fontName)
			return nil
		})
	if err != nil { log.Fatal(err) }

	// add some padding to the computed width and height
	width  += 16
	height += 12

	// create a target image and fill it with white
	outImage := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < width*height*4; i++ { outImage.Pix[i] = 255 }

	// set target and draw each font name
	renderer.SetTarget(outImage)
	sort.Strings(names)
	y := 6
	for _, name := range names {
		renderer.SetFont(fontLib.GetFont(name)) // select the proper font
		h := renderer.SelectionRect(name).Height.Ceil()
		y += h/2 // advance half of the line height
		renderer.Draw(name, width/2, y) // draw font centered
		y += h - h/2 // advance remaining line height
	}

	// store image as png
	filename, err := filepath.Abs("gtxt_each_font.png")
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
