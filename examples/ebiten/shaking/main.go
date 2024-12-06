package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/sfntshape"
	"golang.org/x/image/font/sfnt"
)

// mmmmm... no, this wasn't social commentary on parenting

type Game struct {
	text       *etxt.Renderer
	childX     float64
	parentX    float64
	parent     *ebiten.Image
	child      *ebiten.Image
	shakeLevel fract.Unit
	// ...add a variable for prolonged trauma?
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth := int(math.Ceil(float64(winWidth) * scale))
	canvasHeight := int(math.Ceil(float64(winHeight) * scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	// make child move towards parent
	scale := ebiten.DeviceScaleFactor()
	parentX, _ := ebiten.CursorPosition()
	self.parentX = float64(parentX)
	if self.parentX > self.childX {
		dist := self.parentX - self.childX
		if dist > 22*scale {
			self.childX += scale
		}
	} else if self.parentX < self.childX {
		dist := self.childX - self.parentX
		if dist > 22*scale {
			self.childX -= scale
		}
	}
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	scale := ebiten.DeviceScaleFactor()

	// dark background
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// get shake level
	shakeLevel := self.parentX - self.childX
	if shakeLevel < 0 {
		shakeLevel = -shakeLevel
	}
	if shakeLevel >= 22*scale {
		shakeLevel -= 22 * scale
	} else {
		shakeLevel = 0
	}
	shakeLevel = shakeLevel / 16
	if shakeLevel > 0.2 {
		self.shakeLevel = fract.FromFloat64(shakeLevel + 1)
	} else {
		self.shakeLevel = 1
	}

	// draw both parent and child
	w, h := screen.Size()
	opts := ebiten.DrawImageOptions{}
	bounds := self.parent.Bounds()
	opts.GeoM.Translate(self.parentX+float64(bounds.Min.X), float64(h+bounds.Min.Y))
	screen.DrawImage(self.parent, &opts)
	opts.GeoM.Reset()
	bounds = self.child.Bounds()
	opts.GeoM.Translate(self.childX+float64(bounds.Min.X), float64(h+bounds.Min.Y))
	screen.DrawImage(self.child, &opts)

	// draw text
	self.text.Draw(screen, "I'm not afraid!", w/2, h/2)
}

// This is the key part for this example. It's a custom glyph drawing
// function that we set directly for our text renderer and allows us
// to create a shaking effect. Notice that the shaking is not even
// centered, it's intentionally very messy.
func (self *Game) drawShakyGlyph(target *ebiten.Image, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
	origin.X += fract.Unit(rand.Intn(int(self.shakeLevel)))
	origin.Y += fract.Unit(rand.Intn(int(self.shakeLevel)))
	mask := self.text.Glyph().LoadMask(glyphIndex, origin)
	self.text.Glyph().DrawMask(target, mask, origin)
}

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetColor(color.RGBA{255, 255, 255, 255}) // white
	renderer.SetFont(sfntFont)
	renderer.SetSize(58)
	renderer.SetAlign(etxt.Center)

	// create a game struct and override the renderer's
	// glyph drawing function with one of its methods
	game := Game{text: renderer, parentX: 500, childX: 500}
	renderer.Glyph().SetDrawFunc(game.drawShakyGlyph)

	// create the parent (human) shape with a trapezoid
	// and a circle (well, some lazy round shape)
	shape := sfntshape.New()
	shape.SetScale(ebiten.DeviceScaleFactor())
	shape.MoveTo(-2, 16) // move to the top of the trapezoid
	shape.LineTo(2, 16)
	shape.LineTo(7, -8)
	shape.LineTo(-7, -8)
	shape.LineTo(-2, 16)         // close the trapezoid
	shape.MoveTo(0, 16)          // move to the start of the circle (bottom)
	shape.QuadTo(-8, 16, -8, 24) // draw first quarter (to left-middle)
	shape.QuadTo(-8, 32, 0, 32)  // draw second quarter (to top)
	shape.QuadTo(8, 32, 8, 24)   // draw third quarter (to right-middle)
	shape.QuadTo(8, 16, 0, 16)   // close the shape
	white := color.RGBA{255, 255, 255, 255}
	trans := color.RGBA{0, 0, 0, 0}
	img := shape.Paint(white, trans)
	opts := &ebiten.NewImageFromImageOptions{PreserveBounds: true}
	game.parent = ebiten.NewImageFromImageWithOptions(img, opts)

	// create the child, which is the same but smaller
	shape.Reset()
	shape.MoveTo(-2, 10) // move to the top of the trapezoid
	shape.LineTo(2, 10)
	shape.LineTo(5, -6)
	shape.LineTo(-5, -6)
	shape.LineTo(-2, 10)         // close the trapezoid
	shape.MoveTo(0, 10)          // move to the start of the circle (bottom)
	shape.QuadTo(-6, 10, -6, 16) // draw first quarter (to left-middle)
	shape.QuadTo(-6, 22, 0, 22)  // draw second quarter (to top)
	shape.QuadTo(6, 22, 6, 16)   // draw third quarter (to right-middle)
	shape.QuadTo(6, 10, 0, 10)   // close the shape
	gray := color.RGBA{128, 128, 128, 255}
	img = shape.Paint(gray, trans)
	game.child = ebiten.NewImageFromImageWithOptions(img, opts)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/shaking")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&game)
	if err != nil {
		log.Fatal(err)
	}
}
