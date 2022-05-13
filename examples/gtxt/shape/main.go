//go:build gtxt

package main

import "os"
import "log"
import "fmt"
import "image/color"
import "image/png"
import "path/filepath"

import "github.com/tinne26/etxt/emask"

// An example of how to use emask.Shape in isolation. This has nothing
// to do with fonts, but it can still come in handy in games. There's
// actually another, more practical and simpler example of emask.Shape
// within examples/ebiten/shaking. There's also a crazier random shape
// generator in examples/ebiten/rng_shape.

func main() {
	// create a new shape, preallocating a buffer with capacity
	// for at least 128 segments so we won't have extra allocations
	// later. not like it matters much for such a small example.
	shape := emask.NewShape(128)

	// draw a diamond shape repeatedly, each time smaller.
	// inverting the y each time leads each diamond to be
	// drawn in a different order (clockwise/counter-clockwise),
	// which leads to the shape being patterned on and off
	for i := 0; i < 20; i += 2 {
		shape.InvertY(!shape.HasInvertY()) // comment to disable the pattern
		x := 80 - i
		shape.MoveTo( 0,  x)
		shape.LineTo( x,  0)
		shape.LineTo( 0, -x)
		shape.LineTo(-x,  0)
		shape.LineTo( 0,  x)
	}

	// try uncommenting for a weird effect
	//shape.InvertY(!shape.HasInvertY())

	// draw a few concentric squares
	for i := 0; i < 20; i += 4 {
		shape.InvertY(!shape.HasInvertY())
		x := 50 - i
		shape.MoveTo(-x,  x)
		shape.LineTo( x,  x)
		shape.LineTo( x, -x)
		shape.LineTo(-x, -x)
		shape.LineTo(-x,  x)
	}

	// use the handy Paint function to go from an alpha mask
	// to a RGBA image. in general if you are feeling less
	// fancy you simply use (color.White, color.Black)
	emerald := color.RGBA{  80, 200, 120, 255 }
	catawba := color.RGBA{ 119,  51,  68, 255 }
	outImage := shape.Paint(emerald, catawba)

	// print the path where we will store the result
	filename, err := filepath.Abs("gtxt_shape.png")
	if err != nil { log.Fatal(err) }
	fmt.Printf("Output image: %s\n", filename)

	// actually store the result
	file, err := os.Create(filename)
	if err != nil { panic(err) }
	err = png.Encode(file, outImage)
	if err != nil { panic(err) }

	// bye bye
	fmt.Print("Program exited successfully.\n")
}
