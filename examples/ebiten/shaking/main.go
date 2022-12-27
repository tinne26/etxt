package main

import "os"
import "log"
import "fmt"
import "time"
import "math"
import "image/color"
import "math/rand"

import "github.com/hajimehoshi/ebiten/v2"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/emask"

// mmmmm... no, this wasn't social commentary on parenting

type Game struct {
	txtRenderer *etxt.Renderer
	childX float64
	parentX float64
	parent *ebiten.Image
	child *ebiten.Image
	// ...add a variable for prolonged trauma?
}

func (self *Game) Layout(w int, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	return int(math.Ceil(float64(w)*scale)), int(math.Ceil(float64(h)*scale))
}
func (self *Game) Update() error {
	scale := ebiten.DeviceScaleFactor()
	parentX, _ := ebiten.CursorPosition()
	self.parentX = float64(parentX)
	if self.parentX > self.childX {
		dist := self.parentX - self.childX
		if dist > 22*scale { self.childX += scale }
	} else if self.parentX < self.childX {
		dist := self.childX - self.parentX
		if dist > 22*scale { self.childX -= scale }
	}
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	scale := ebiten.DeviceScaleFactor()

	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// get shake level
	shakeLevel := self.parentX - self.childX
	if shakeLevel < 0 { shakeLevel = -shakeLevel }
	if shakeLevel >= 22*scale { shakeLevel -= 22*scale } else { shakeLevel = 0 }
	shakeLevel = shakeLevel/16

	// draw parent
	w, h := screen.Size()
	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Translate(self.parentX - 6*scale, float64(h) - 32*scale)
	screen.DrawImage(self.parent, &opts)

	// draw children
	opts  = ebiten.DrawImageOptions{}
	opts.GeoM.Translate(self.childX - 5*scale, float64(h) - 20*scale)
	opts.ColorM.Scale(0.5, 0.5, 0.5, 1.0)
	screen.DrawImage(self.child, &opts)

	// draw text
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.Traverse("I'm not afraid!", fixed.P(w/2, h/2),
		func(dot fixed.Point26_6, _ rune, glyphIndex etxt.GlyphIndex) {
			if shakeLevel > 0 {
				dot.X += fixed.Int26_6(rand.Intn(int(shakeLevel) + 1)*64)
				dot.Y += fixed.Int26_6(rand.Intn(int(shakeLevel) + 1)*64)
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
	scale := ebiten.DeviceScaleFactor()
	renderer := etxt.NewStdRenderer()
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(int(58*scale))
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white

	// create the parent (human) shape with a trapezoid and a circle
	expectedParts := 10
	shape := emask.NewShape(expectedParts)
	moveTo(&shape, -2, 24, scale) // move to the top of the trapezoid
	lineTo(&shape, -7,  0, scale)
	lineTo(&shape,  7,  0, scale)
	lineTo(&shape,  2, 24, scale)
	lineTo(&shape, -2, 24, scale) // close the trapezoid
	moveTo(&shape,  0, 24, scale) // move to the start of the circle (bottom)
	quadTo(&shape, -8, 24, -8, 32, scale)// draw first quarter (to left-middle)
	quadTo(&shape, -8, 40,  0, 40, scale)// draw second quarter (to top)
	quadTo(&shape,  8, 40,  8, 32, scale)// draw third quarter (to right-middle)
	quadTo(&shape,  8, 24,  0, 24, scale)// close the shape
	pixelAligned := fixed.Point26_6{}
	mask, err := emask.Rasterize(shape.Segments(), renderer.GetRasterizer(), pixelAligned)
	if err != nil { log.Fatal(err) }
	parentImg := ebiten.NewImageFromImage(mask) // *
	// * Notice that Ebitengine won't preserve the mask bounds, so we won't be
	//   able to use them to position the image... we will do it by hand.

	// create the kid, which is the same but smaller
	shape.Reset()
	moveTo(&shape, -2, 16, scale) // move to the top of the trapezoid
	lineTo(&shape, -5,  0, scale)
	lineTo(&shape,  5,  0, scale)
	lineTo(&shape,  2, 16, scale)
	lineTo(&shape, -2, 16, scale) // close the trapezoid
	moveTo(&shape,  0, 16, scale) // move to the start of the circle (bottom)
	quadTo(&shape, -6, 16, -6, 22, scale)// draw first quarter (to left-middle)
	quadTo(&shape, -6, 28,  0, 28, scale)// draw second quarter (to top)
	quadTo(&shape,  6, 28,  6, 22, scale)// draw third quarter (to right-middle)
	quadTo(&shape,  6, 16,  0, 16, scale)// close the shape
	mask, err = emask.Rasterize(shape.Segments(), renderer.GetRasterizer(), pixelAligned)
	if err != nil { log.Fatal(err) }
	childImg := ebiten.NewImageFromImage(mask)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/shaking")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game { renderer, 500, 500, parentImg, childImg })
	if err != nil { log.Fatal(err) }
}

func moveTo(shape *emask.Shape, x, y int, scale float64) {
	sx, sy := int(float64(x)*scale), int(float64(y)*scale)
	shape.MoveTo(sx, sy)
}

func lineTo(shape *emask.Shape, x, y int, scale float64) {
	sx, sy := int(float64(x)*scale), int(float64(y)*scale)
	shape.LineTo(sx, sy)
}

func quadTo(shape *emask.Shape, ctrlX, ctrlY, x, y int, scale float64) {
	scx, scy := int(float64(ctrlX)*scale), int(float64(ctrlY)*scale)
	sx , sy  := int(float64(x)*scale), int(float64(y)*scale)
	shape.QuadTo(scx, scy, sx, sy)
}
