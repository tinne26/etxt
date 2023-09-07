//go:build GENERATE_ETXT_TESTDATA && gtxt

package main

import "os"
import "fmt"
import "image"
import "math/rand"
import "strconv"
import "image/color"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/fract"

// See etxt/testdata_generate.go for details.
// Must be generated from base etxt directory, so testdata
// files are placed at the same level as testdata_generate.go.

var contents []byte = []byte("package etxt\n\nfunc init() {\n\ttestdata[\"blend_rand_gtxt\"] = []byte{")

func main() {
	const filename = "testdata_blend_rand_gtxt_test.go"
	fmt.Print("Generating '" + filename + "'... ")

	// draw, obtain result values, encode them
	renderer := etxt.NewRenderer()
	target := image.NewRGBA(image.Rect(0, 0, 8, 8))
	fill(target, color.RGBA{96, 96, 96, 96})
	mask := image.NewAlpha(image.Rect(0, 0, 1, 1))
	mask.Set(0, 0, color.Alpha{255})
	rng := rand.New(rand.NewSource(3707))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			a := rng.Intn(256)
			r, g, b := rng.Intn(a + 1), rng.Intn(a + 1), rng.Intn(a + 1)
			rngColor := color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
			renderer.SetColor(rngColor)
			renderer.Glyph().DrawMask(target, mask, fract.IntsToPoint(x, y))
		}
	}
	
	for i, value := range target.Pix {
		if i % 32 == 0 {
			contents = append(contents, '\n', '\t', '\t')
		} else if i % 4 == 0 {
			contents = append(contents, '/', '*', '*', '/', ' ')
		}
		contents = append(contents, []byte(strconv.Itoa(int(value)))...)
		contents = append(contents, ',', ' ')
	}
	contents = append(contents, []byte("\n\t}\n}\n")...)

	file, err := os.Create(filename)
	if err != nil { fatal(err) }
	_, err = file.Write(contents)
	if err != nil {
		_ = os.Remove(filename)
		fatal(err)
	}

	fmt.Print("OK\n")
}

func fatal(err error) {
	fmt.Fprint(os.Stderr, "\nERROR: " + err.Error() + "\n")
	os.Exit(1)
}

func fill(img *image.RGBA, clr color.RGBA) {
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			img.SetRGBA(x, y, clr)
		}
	}
}
