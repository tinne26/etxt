package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
	"github.com/tinne26/etxt/fract"
)

// This example is an interactive demo for testing text aligns.
// You can select a string, coordinate and align to see how the
// rendering will look. You can run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/aligns@latest path/to/font.ttf

type Game struct {
	text        *etxt.Renderer
	contentType int
	direction   etxt.Direction
	align       etxt.Align
	maxWrapLen  int
	x, y        int
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth := int(math.Ceil(float64(winWidth) * scale))
	canvasHeight := int(math.Ceil(float64(winHeight) * scale))
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

	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if self.direction == etxt.LeftToRight {
			self.direction = etxt.RightToLeft
		} else {
			self.direction = etxt.LeftToRight
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		self.x, self.y = ebiten.CursorPosition()
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		x, _ := ebiten.CursorPosition()
		newWrapLen := self.x - x
		if newWrapLen < 0 {
			newWrapLen = -newWrapLen
		}
		if self.align.Horz() == etxt.HorzCenter {
			newWrapLen *= 2
		}
		if newWrapLen == 0 {
			newWrapLen = 1
		}
		if newWrapLen == self.maxWrapLen {
			newWrapLen = 0 // disable wrapping
		}
		self.maxWrapLen = newWrapLen
	}

	return nil
}

const NumContentTypes = 5

func (self *Game) Draw(canvas *ebiten.Image) {
	// dark background, wrap area and position lines
	bounds := canvas.Bounds()
	scale := ebiten.DeviceScaleFactor()
	w, h := bounds.Dx(), bounds.Dy()
	canvas.Fill(color.RGBA{3, 2, 0, 255})
	if self.maxWrapLen > 0 {
		leftOffset, rightOffset := self.getWrapOffsets()
		area := fract.IntsToRect(self.x+leftOffset, 0, self.x+rightOffset, h).Clip(canvas)
		area.Fill(color.RGBA{16, 26, 26, 255})
	}
	line := fract.IntsToRect(self.x, 0, self.x+1, h).Clip(canvas)
	line.Fill(color.RGBA{32, 32, 18, 255})
	line = fract.IntsToRect(0, self.y, w, self.y+1).Clip(canvas)
	line.Fill(color.RGBA{32, 32, 18, 255})

	// draw helper text
	pad := int(10 * scale)
	info := "[H] Horz. Align " + self.align.Horz().String() + "\n"
	info += "[V] Vert. Align " + self.align.Vert().String() + "\n"
	info += "[T] Text type\n"
	info += "[D] Text direction (" + self.direction.String() + ")\n"
	info += "(Left click to change drawing coords, right click to adjust line wrapping)"
	self.text.Draw(canvas, info, pad, h-pad)

	// draw aligned text
	self.text.Utils().AssertMaxStoredStates(0)
	self.text.Utils().StoreState()
	defer self.text.Utils().RestoreState()

	self.text.SetSize(18)
	self.text.SetColor(color.RGBA{255, 255, 255, 255})
	self.text.SetAlign(self.align)
	self.text.SetDirection(self.direction)
	var content string
	switch self.contentType {
	case 0: // align
		content = self.align.String()
	case 1: // multiline
		content = "she always saw\nthrough the eyes of others\nas if they were her own"
	case 2: // uppercase
		content = "STOP SHOUTING LIKE THAT!"
	case 3:
		content = "ABCDEFGHI\nJKLMNOPQR\nSTUVWXYZ"
	case 4:
		content = "\\^_^/"
	}
	if self.maxWrapLen > 0 {
		self.text.DrawWithWrap(canvas, content, self.x, self.y, self.maxWrapLen)
	} else {
		self.text.Draw(canvas, content, self.x, self.y)
	}
}

// Helper function for drawing the text wrap area.
func (self *Game) getWrapOffsets() (left, right int) {
	switch self.align.Horz() {
	case etxt.HorzCenter:
		return -self.maxWrapLen / 2, +self.maxWrapLen / 2
	case etxt.Left:
		return 0, self.maxWrapLen
	case etxt.Right:
		return -self.maxWrapLen, 0
	default:
		panic(self.align.Horz())
	}
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
	renderer.SetFont(sfntFont)
	renderer.SetColor(color.RGBA{128, 128, 128, 255})
	renderer.SetAlign(etxt.LastBaseline | etxt.Left)
	renderer.SetSize(15)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/aligns")
	ebiten.SetWindowSize(640, 480)
	scale := ebiten.DeviceScaleFactor()
	err = ebiten.RunGame(&Game{
		text:      renderer,
		align:     etxt.Center,
		direction: etxt.LeftToRight,
		x:         int(320 * scale),
		y:         int(240 * scale),
	})
	if err != nil {
		log.Fatal(err)
	}
}

// --- helper code for aligns ---

var horzAligns = []etxt.Align{etxt.Left, etxt.HorzCenter, etxt.Right}
var vertAligns = []etxt.Align{
	etxt.Top, etxt.CapLine, etxt.Midline, etxt.VertCenter, etxt.Baseline,
	etxt.Bottom, etxt.LastBaseline,
}

func nextAlign(aligns []etxt.Align, align etxt.Align) etxt.Align {
	for n, nthAlign := range aligns {
		if nthAlign != align {
			continue
		}
		return aligns[(n+1)%len(aligns)]
	}
	panic("failed to find next align")
}

func prevAlign(aligns []etxt.Align, align etxt.Align) etxt.Align {
	for n, nthAlign := range aligns {
		if nthAlign != align {
			continue
		}
		return aligns[n-1%len(aligns)]
	}
	panic("failed to find previous align")
}
