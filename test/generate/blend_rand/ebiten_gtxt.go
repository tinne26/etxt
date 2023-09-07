//go:build GENERATE_ETXT_TESTDATA && gtxt

package main

import "os"
import "fmt"
import "math/rand"
import "strconv"
import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/fract"

// See etxt/testdata_generate.go for details.
// Must be generated from base etxt directory, so testdata
// files are placed at the same level as testdata_generate.go.

var contents []byte = []byte("package etxt\n\nfunc init() {\n\ttestdata[\"blend_rand_ebiten_gtxt\"] = []byte{")

type Game struct {}
func (self *Game) Layout(w, h int) (int, int) { return w, h }
func (self *Game) Draw(*ebiten.Image) {}
func (self *Game) Update() error {
	renderer := etxt.NewRenderer()
	target := ebiten.NewImage(8, 8)
	target.Fill(color.RGBA{96, 96, 96, 96})
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
	
	buffer := make([]byte, 8*8*4)
	target.ReadPixels(buffer)
	for i, value := range buffer {
		if i % 32 == 0 {
			contents = append(contents, '\n', '\t', '\t')
		} else if i % 4 == 0 {
			contents = append(contents, '/', '*', '*', '/', ' ')
		}
		contents = append(contents, []byte(strconv.Itoa(int(value)))...)
		contents = append(contents, ',', ' ')
	}
	contents = append(contents, []byte("\n\t}\n}\n")...)

	return ebiten.Termination
}

func main() {
	const filename = "testdata_blend_rand_ebiten_gtxt_test.go"
	fmt.Print("Generating '" + filename + "'... ")
	
	err := ebiten.RunGame(&Game{})
	if err != nil { fatal(err) }

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
