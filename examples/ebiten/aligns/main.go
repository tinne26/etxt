package main

import "os"
import "log"
import "fmt"
import "math"
import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2/inpututil"
import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"

// This example is an interactive demo for testing text aligns.
// You can select a string, coordinate and align to see how the
// rendering will look. You can run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/aligns@latest path/to/font.ttf

type Game struct {
	text *etxt.Renderer
	contentType int
	align etxt.Align
	x, y int
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)

	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		var newHorzAlign etxt.Align
		if shift {
			newHorzAlign = prevAlign(horzAligns, self.align.Horz())
		} else {
			newHorzAlign = nextAlign(horzAligns, self.align.Horz())
		}
		self.align = self.align.Adjusted(newHorzAlign)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		var newVertAlign etxt.Align
		if shift {
			newVertAlign = prevAlign(vertAligns, self.align.Vert())
		} else {
			newVertAlign = nextAlign(vertAligns, self.align.Vert())
		}
		self.align = self.align.Adjusted(newVertAlign)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		if shift {
			self.contentType = (self.contentType + NumContentTypes - 1) % NumContentTypes
		} else {
			self.contentType = (self.contentType + 1) % NumContentTypes
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		self.x, self.y = ebiten.CursorPosition()
	}

	return nil
}

const NumContentTypes = 5
func (self *Game) Draw(canvas *ebiten.Image) {
	// dark background and position lines
	bounds := canvas.Bounds()
	scale := ebiten.DeviceScaleFactor()
	w, h := bounds.Dx(), bounds.Dy()
	canvas.Fill(color.RGBA{ 3, 2, 0, 255 })
	line := canvas.SubImage(image.Rect(self.x, 0, self.x + 1, h)).(*ebiten.Image)
	line.Fill(color.RGBA{ 32, 32, 18, 255 })
	line  = canvas.SubImage(image.Rect(0, self.y, w, self.y + 1)).(*ebiten.Image)
	line.Fill(color.RGBA{ 32, 32, 18, 255 })

	// draw helper text
	pad := int(10*scale)
	info := "[H] Horz. Align " + self.align.Horz().String() + "\n"
	info += "[V] Vert. Align " + self.align.Vert().String() + "\n"
	info += "[T] Text type\n"
	info += "(Click anywhere to change drawing coordinates)"
	self.text.Draw(canvas, info, pad, h - pad)

	// draw aligned text
	self.text.Utils().StoreState()
	defer self.text.Utils().RestoreState()

	self.text.SetSize(18)
	self.text.SetColor(color.RGBA{255, 255, 255, 255})
	self.text.SetAlign(self.align)
	var content string
	switch self.contentType {
	case 0: // align
		content = self.align.String()
	case 1: // multiline
		content = "the word for seeing\nthrough the eyes of others\nas if they were your own"
	case 2: // uppercase
		content = "STOP SHOUTING LIKE THAT!"
	case 3:
		content = "ABCDEFGHI\nJKLMNOPQR\nSTUVWXYZ"
	case 4:
		content = "\\^_^/"
	}
	self.text.Draw(canvas, content, self.x, self.y)
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
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetFont(sfntFont)
	renderer.SetColor(color.RGBA{128, 128, 128, 255}) // white
	renderer.SetAlign(etxt.LastBaseline | etxt.Left)
	renderer.SetSize(15)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/aligns")
	ebiten.SetWindowSize(640, 480)
	scale := ebiten.DeviceScaleFactor()
	err = ebiten.RunGame(&Game{
		text: renderer,
		align: etxt.Center,
		x: int(320*scale),
		y: int(240*scale),
	})
	if err != nil { log.Fatal(err) }
}

// --- helper code for aligns ---

var horzAligns = []etxt.Align{ etxt.Left, etxt.HorzCenter, etxt.Right }
var vertAligns = []etxt.Align{
	etxt.Top, etxt.Midline, etxt.VertCenter, etxt.Baseline,
	etxt.Bottom, etxt.LastMidline, etxt.LastBaseline,
}

func nextAlign(aligns []etxt.Align, align etxt.Align) etxt.Align {
	for n, nthAlign := range aligns {
		if nthAlign != align { continue }
		return aligns[(n + 1) % len(aligns)]
	}
	panic("failed to find next align")
}

func prevAlign(aligns []etxt.Align, align etxt.Align) etxt.Align {
	for n, nthAlign := range aligns {
		if nthAlign != align { continue }
		return aligns[n - 1 % len(aligns)]
	}
	panic("failed to find previous align")
}
