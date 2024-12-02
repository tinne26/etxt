package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
	"github.com/tinne26/etxt/fract"
	"golang.org/x/image/font/sfnt"
)

// This example showcases how to use the advanced custom draw
// and line change functions to implement color changes through
// simple string markup.
//
// You can run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/color_markup@latest path/to/font.ttf

const Content = "You can use [rgb(0 255 255)]{rgb(R, G, B)} notation to change the text color, " +
	"but you can't do recursive wrapping, only simple changes like turning text [rgb(255 0 0)]{red} " +
	"or [rgb(255 255 0)]{yellow} or whatever."

type Game struct {
	text         *etxt.Renderer
	direction    etxt.Direction
	align        etxt.Align
	content      string
	maxLineLen   int
	clrProcessor ColorProcessor
	x, y         int
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

	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if self.direction == etxt.LeftToRight {
			self.direction = etxt.RightToLeft
		} else {
			self.direction = etxt.LeftToRight
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		self.x, self.y = ebiten.CursorPosition()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		x, _ := ebiten.CursorPosition()
		self.maxLineLen = int(math.Abs(2 * float64(x-self.x)))
		if self.maxLineLen < 96 {
			self.maxLineLen = 96
		}
	}

	return nil
}

func (self *Game) Draw(canvas *ebiten.Image) {
	// dark background and position lines
	bounds := canvas.Bounds()
	scale := ebiten.DeviceScaleFactor()
	w, h := bounds.Dx(), bounds.Dy()
	canvas.Fill(color.RGBA{3, 2, 0, 255})
	wrapAreaLeft := self.x - self.maxLineLen/2
	wrapAreaRight := wrapAreaLeft + self.maxLineLen
	sub := canvas.SubImage(image.Rect(wrapAreaLeft, 0, wrapAreaRight, h))
	sub.(*ebiten.Image).Fill(color.RGBA{16, 16, 9, 255})
	line := fract.IntsToRect(self.x, 0, self.x+1, h).Clip(canvas)
	line.Fill(color.RGBA{32, 32, 18, 255})
	line = fract.IntsToRect(0, self.y, w, self.y+1).Clip(canvas)
	line.Fill(color.RGBA{32, 32, 18, 255})

	// draw helper text
	pad := int(10 * scale)
	info := "[H] Horz. Align " + self.align.Horz().String() + "\n"
	info += "[V] Vert. Align " + self.align.Vert().String() + "\n"
	info += "[D] Text direction (" + self.direction.String() + ")\n"
	info += "(Click anywhere to change drawing coordinates)"
	self.text.Draw(canvas, info, pad, h-pad)

	// draw aligned text
	self.text.Utils().AssertMaxStoredStates(0)
	self.text.Utils().StoreState()
	defer self.text.Utils().RestoreState()

	self.text.SetSize(18)
	self.text.SetColor(color.RGBA{255, 255, 255, 255})
	self.text.SetAlign(self.align)
	self.text.SetDirection(self.direction)
	self.clrProcessor.SetContent(self.content)
	self.clrProcessor.DrawWithWrap(self.text, canvas, self.x, self.y, self.maxLineLen)
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
	ebiten.SetWindowTitle("etxt/examples/ebiten/color_markup")
	ebiten.SetWindowSize(640, 480)
	scale := ebiten.DeviceScaleFactor()
	err = ebiten.RunGame(&Game{
		text:       renderer,
		align:      etxt.Center,
		direction:  etxt.LeftToRight,
		content:    Content,
		maxLineLen: 500,
		x:          int(320 * scale),
		y:          int(240 * scale),
	})
	if err != nil {
		log.Fatal(err)
	}
}

// --- color processor ---

type ColorChange struct {
	startIndex int
	endIndex   int
	rgba       color.RGBA
	prevColor  color.Color
}

type ColorProcessor struct {
	rawContent  string
	postContent string
	changes     []ColorChange

	opRenderer   *etxt.Renderer
	opIndex      int
	opChange     int
	opNextEnd    int
	opNextStart  int
	opLastOrigin fract.Point
}

var colorProcessorRegex = regexp.MustCompile(`\[rgb\((\d{1,3}) (\d{1,3}) (\d{1,3})\)\]\{(.*?)\}`)

func (self *ColorProcessor) SetContent(rawContent string) {
	if rawContent == self.rawContent {
		return
	}

	self.changes = self.changes[:0]
	self.rawContent = rawContent
	self.postContent = rawContent
	for {
		submatches := colorProcessorRegex.FindStringSubmatchIndex(self.postContent)
		if submatches == nil {
			return
		}

		// get colors and limit to 255
		r, err := strconv.Atoi(self.postContent[submatches[2]:submatches[3]])
		if err != nil {
			panic(err)
		}
		g, err := strconv.Atoi(self.postContent[submatches[4]:submatches[5]])
		if err != nil {
			panic(err)
		}
		b, err := strconv.Atoi(self.postContent[submatches[6]:submatches[7]])
		if err != nil {
			panic(err)
		}
		if r > 255 {
			r = 255
		}
		if g > 255 {
			g = 255
		}
		if b > 255 {
			b = 255
		}

		// replace text (not very efficient)
		pre := self.postContent[0:submatches[0]]
		inner := self.postContent[submatches[8]:submatches[9]]
		post := self.postContent[submatches[1]:]
		self.postContent = pre + inner + post
		self.changes = append(self.changes, ColorChange{
			startIndex: submatches[0],
			endIndex:   submatches[0] + len(inner),
			rgba:       color.RGBA{uint8(r), uint8(g), uint8(b), 255},
		})
	}
}

func (self *ColorProcessor) DrawWithWrap(renderer *etxt.Renderer, canvas *ebiten.Image, x, y, widthLimit int) {
	// configure for draw
	renderer.Glyph().SetDrawFunc(self.drawFn)
	renderer.Glyph().SetLineChangeFunc(self.lineChangeFn)
	self.opRenderer = renderer
	self.opIndex = 0
	self.opChange = -1
	self.opNextEnd = math.MaxInt
	self.opNextStart = math.MaxInt
	if len(self.changes) > 0 {
		self.opNextStart = self.changes[0].startIndex
		self.opNextEnd = self.changes[0].endIndex
		self.opChange = 0
	}

	// draw
	renderer.DrawWithWrap(canvas, self.postContent, x, y, widthLimit)

	// state clean-up
	self.opRenderer = nil
	renderer.Glyph().SetDrawFunc(nil)
	renderer.Glyph().SetLineChangeFunc(nil)
}

func (self *ColorProcessor) drawFn(canvas *ebiten.Image, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
	self.increaseOpIndex()
	self.opLastOrigin = origin
	mask := self.opRenderer.Glyph().LoadMask(glyphIndex, origin)
	self.opRenderer.Glyph().DrawMask(canvas, mask, origin)
}

func (self *ColorProcessor) lineChangeFn(lineChangeDetails etxt.LineChangeDetails) {
	if lineChangeDetails.ElidedSpace {
		self.increaseOpIndex()
	}
	if !lineChangeDetails.IsWrap {
		self.increaseOpIndex()
	}
}

func (self *ColorProcessor) increaseOpIndex() {
	if self.opNextEnd <= self.opIndex {
		self.opRenderer.SetColor(self.changes[self.opChange].prevColor)
		self.opChange += 1
		if self.opChange < len(self.changes) {
			self.opNextStart = self.changes[self.opChange].startIndex
			self.opNextEnd = self.changes[self.opChange].endIndex
		} else {
			self.opNextStart = math.MaxInt
			self.opNextEnd = math.MaxInt
		}
	}
	if self.opNextStart <= self.opIndex {
		self.changes[self.opChange].prevColor = self.opRenderer.GetColor()
		self.opRenderer.SetColor(self.changes[self.opChange].rgba)
		self.opNextStart = math.MaxInt
	}
	self.opIndex += 1
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
