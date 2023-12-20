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
	renderer.SetSize(18)
	renderer.SetFont(sfntFont)
	renderer.SetColor(color.RGBA{40, 0, 0, 255})

	// create target image and fill it with a dark background color,
	// four rectangles to draw text with different aligns within each
	// one, including guide lines and a central mark for each rectangle
	// (this has nothing to do with etxt, it's only to make it look nice)
	target, targets := makeFancyOutImage()

	// default (Baseline, Left) align
	// renderer.SetAlign(etxt.Baseline | etxt.Left) // (optional line)
	renderer.Draw(target, "(Baseline, Left)", targets[0].X, targets[0].Y)

	// (VertCenter, HorzCenter) align
	renderer.SetAlign(etxt.Center)
	renderer.Draw(target, "(VertCenter, HorzCenter)", targets[1].X, targets[1].Y)

	// (Top, Right) align
	renderer.SetAlign(etxt.Top | etxt.Right)
	renderer.Draw(target, "(Top, Right)", targets[2].X, targets[2].Y)

	// (Bottom, HorzCenter) align
	renderer.SetAlign(etxt.Bottom | etxt.HorzCenter)
	renderer.Draw(target, "(Bottom, HorzCenter)", targets[3].X, targets[3].Y)

	// store image as png
	filename, err := filepath.Abs("gtxt_aligns.png")
	if err != nil { log.Fatal(err) }
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil { log.Fatal(err) }
	err = png.Encode(file, target)
	if err != nil { log.Fatal(err) }
	err = file.Close()
	if err != nil { log.Fatal(err) }
	fmt.Print("Program exited successfully.\n")
}

// Creates an image with four subrectangles in it, each with guide lines
// and a mark at their center, so we can use it to draw with different
// aligns on top and see how they relate to the given marks.
//
// This has nothing to do with etxt itself, so you don't need to understand
// it, and if you are doing game dev this will be trivial for you anyway.
func makeFancyOutImage() (*image.RGBA, [4]image.Point) {
	// out image properties
	rectWidth  := 301
	rectHeight := 101
	padding    := 4
	backColor  := color.RGBA{ R: 236, G: 236, B: 230, A: 255 }
	rectColor  := color.RGBA{ R: 200, G: 196, B: 206, A: 255 }
	guideColor := color.RGBA{ R: 220, G: 220, B: 220, A: 255 }
	markColor  := color.RGBA{ R:   0, G:  80, B: 120, A: 255 }
	markColor2 := color.RGBA{ R:   0, G: 190, B:  80, A: 255 }
	totalWidth  := rectWidth*2  + padding*3
	totalHeight := rectHeight*2 + padding*3
	outImage := image.NewRGBA(image.Rect(0, 0, totalWidth, totalHeight))

	// paint background
	for y := 0; y < totalHeight; y++ {
		for x := 0; x < totalWidth; x++ {
			outImage.Set(x, y, backColor)
		}
	}

	// paint rects
	for y := 0; y < rectHeight; y++ {
		for x := 0; x < rectWidth; x++ {
			// we draw the four rects at once (not ideal for memory accesses)
			outImage.Set(x + padding, y + padding, rectColor)
			outImage.Set(x + padding*2 + rectWidth, y + padding, rectColor)
			lowerY := y + rectHeight + padding*2
			outImage.Set(x + padding, lowerY, rectColor)
			outImage.Set(x + padding*2 + rectWidth, lowerY, rectColor)
		}
	}

	// paint guide lines
	for x := 0; x < rectWidth; x++ { // horizontal guide lines
		y := padding + rectHeight/2
		outImage.Set(x + padding, y, guideColor)
		outImage.Set(x + padding*2 + rectWidth, y, guideColor)
		yBottom := y + padding + rectHeight
		outImage.Set(x + padding, yBottom, guideColor)
		outImage.Set(x + padding*2 + rectWidth, yBottom, guideColor)
	}
	for y := 0; y < rectHeight; y++ { // vertical guide lines
		outImage.Set(padding + rectWidth/2, y + padding, guideColor)
		outImage.Set(padding + rectWidth/2, y + padding*2 + rectHeight, guideColor)
		xRight  := rectWidth/2 + padding*2 + rectWidth
		outImage.Set(xRight, y + padding, guideColor)
		outImage.Set(xRight, y + padding*2 + rectHeight, guideColor)
	}

	// create target points for reference marks
	ta := image.Pt(rectWidth/2 + padding, rectHeight/2 + padding)
	tb := image.Pt(rectWidth/2 + padding*2 + rectWidth, rectHeight/2 + padding)
	tc := image.Pt(rectWidth/2 + padding, rectHeight/2 + padding*2 + rectHeight)
	td := image.Pt(rectWidth/2 + padding*2 + rectWidth, rectHeight/2 + padding*2 + rectHeight)

	// paint reference marks
	drawMarkAt := func (x, y int) {
		outImage.Set(x, y, markColor)
		for i := 1; i < 3; i++ {
			outImage.Set(x + i, y, markColor)
			outImage.Set(x - i, y, markColor)
			outImage.Set(x, y - i, markColor)
			outImage.Set(x, y + i, markColor)
		}
		outImage.Set(x + 1, y + 1, markColor2)
		outImage.Set(x + 1, y - 1, markColor2)
		outImage.Set(x - 1, y + 1, markColor2)
		outImage.Set(x - 1, y - 1, markColor2)
	}
	drawMarkAt(ta.X, ta.Y)
	drawMarkAt(tb.X, tb.Y)
	drawMarkAt(tc.X, tc.Y)
	drawMarkAt(td.X, td.Y)

	return outImage, [4]image.Point{ta, tb, tc, td}
}
