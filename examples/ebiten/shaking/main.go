package main

import "os"
import "log"
import "fmt"
import "time"
import "image/color"
import "math/rand"

import "github.com/hajimehoshi/ebiten/v2"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/emask"

// mmmmm... no, this wasn't social commentary on parenting

type Game struct {
	txtRenderer *etxt.Renderer
	childX int
	parentX int
	parent *ebiten.Image
	child *ebiten.Image
	// ...add a variable for prolonged trauma?
}

func (self *Game) Layout(w int, h int) (int, int) { return w, h }
func (self *Game) Update() error {
	self.parentX, _ = ebiten.CursorPosition()
	if self.parentX > self.childX {
		dist := self.parentX - self.childX
		if dist > 22 { self.childX += 1 }
	} else if self.parentX < self.childX {
		dist := self.childX - self.parentX
		if dist > 22 { self.childX -= 1 }
	}
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// get shake level
	shakeLevel := self.parentX - self.childX
	if shakeLevel < 0 { shakeLevel = -shakeLevel }
	if shakeLevel >= 22 { shakeLevel -= 22 } else { shakeLevel = 0 }
	shakeLevel = shakeLevel/16

	// draw parent
	w, h := screen.Size()
	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(self.parentX - 6), float64(h - 32))
	screen.DrawImage(self.parent, &opts)

	// draw children
	opts  = ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(self.childX - 5), float64(h - 20))
	opts.ColorM.Scale(0.5, 0.5, 0.5, 1.0)
	screen.DrawImage(self.child, &opts)

	// draw text
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.Traverse("I'm not afraid!", fixed.P(w/2, h/2),
		func(dot fixed.Point26_6, _ rune, glyphIndex etxt.GlyphIndex) {
			if shakeLevel > 0 {
				dot.X += fixed.Int26_6(rand.Intn(shakeLevel + 1)*64)
				dot.Y += fixed.Int26_6(rand.Intn(shakeLevel + 1)*64)
			}
			mask := self.txtRenderer.LoadGlyphMask(glyphIndex, dot)
			self.txtRenderer.DefaultDrawFunc(dot, mask, glyphIndex)
		})
}

func main() {
	// seed rand
	rand.Seed(time.Now().UnixNano())

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
	renderer.SetSizePx(58)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white

	// create the parent (human) shape with a trapezoid and a circle
	expectedParts := 10
	shape := emask.NewShape(expectedParts)
	shape.MoveTo(-2, 24) // move to the top of the trapezoid
	shape.LineTo(-7,  0)
	shape.LineTo( 7,  0)
	shape.LineTo( 2, 24)
	shape.LineTo(-2, 24) // close the trapezoid
	shape.MoveTo(0, 24) // move to the start of the circle (bottom)
	shape.QuadTo(-8, 24, -8, 32)// draw first quarter (to left-middle)
	shape.QuadTo(-8, 40,  0, 40)// draw second quarter (to top)
	shape.QuadTo( 8, 40,  8, 32)// draw third quarter (to right-middle)
	shape.QuadTo( 8, 24,  0, 24)// close the shape
	pixelAligned := fixed.Point26_6{}
	mask, err := emask.Rasterize(shape.Segments(), renderer.GetRasterizer(), pixelAligned)
	if err != nil { log.Fatal(err) }
	parentImg := ebiten.NewImageFromImage(mask) // *
	// * Notice that Ebiten won't preserve the mask bounds, so we won't be
	//   able to use them to position the image... we will do it by hand.

	// create the kid, which is the same but smaller
	shape.Reset()
	shape.MoveTo(-2, 16) // move to the top of the trapezoid
	shape.LineTo(-5,  0)
	shape.LineTo( 5,  0)
	shape.LineTo( 2, 16)
	shape.LineTo(-2, 16) // close the trapezoid
	shape.MoveTo(0, 16) // move to the start of the circle (bottom)
	shape.QuadTo(-6, 16, -6, 22)// draw first quarter (to left-middle)
	shape.QuadTo(-6, 28,  0, 28)// draw second quarter (to top)
	shape.QuadTo( 6, 28,  6, 22)// draw third quarter (to right-middle)
	shape.QuadTo( 6, 16,  0, 16)// close the shape
	mask, err = emask.Rasterize(shape.Segments(), renderer.GetRasterizer(), pixelAligned)
	if err != nil { log.Fatal(err) }
	childImg := ebiten.NewImageFromImage(mask)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/shaking")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game { renderer, 500, 500, parentImg, childImg })
	if err != nil { log.Fatal(err) }
}
