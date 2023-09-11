package main

import "os"
import "log"
import "fmt"
import "time"
import "math"
import "math/rand"
import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/inpututil"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"

// This example shows how to create a cutout effect, where instead
// of filling the text with a specific color, we create a see-through
// effect. It also serves as an example of how to center and draw
// glyphs manually with Renderer.Glyph().DrawMask(). There's also
// some basic animation to make it all more entertaining to watch.
//
// - You can press B to toggle the front layer and see the background
//   text in isolation.
// - You can press G to toggle some poor glitch effects.
//
// Notice that this program will look very different with thick and
// slim fonts (artsy with the slim ones, nerdy with the thick ones).
// Try out different ones!
//
// You can run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/cutout@latest path/to/font.ttf

const MainText = "COMPLETE\nSYSTEM\nFAILURE"
const MainFontSize = 94

var runePool = []rune {
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o',
	'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'?', '!', '#', '=', '+', '/', '&',
}

type Game struct {
	backRenderer *etxt.Renderer
	frontRenderer *etxt.Renderer
	backLines [][]rune
	offscreen *ebiten.Image
	backOnly bool
	glitchRects [2]image.Rectangle
	glitchesEnabled bool
	lastWidth int
	lastHeight int
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.backRenderer.SetScale(scale) // relevant for HiDPI
	self.frontRenderer.SetScale(scale)
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	self.lastWidth, self.lastHeight = canvasWidth, canvasHeight
	return canvasWidth, canvasHeight
}

func (self *Game) ensureBackLinesContent() {
	const HeightFactor = 0.9
	const WidthFactor  = 1.2

	buffer := self.backRenderer.GetBuffer()
	font   := self.backRenderer.GetFont()
	size   := self.backRenderer.Fract().GetScaledSize()
	sizer  := self.backRenderer.GetSizer()
	lineHeight := sizer.LineHeight(font, buffer, size)
	mIndex := self.backRenderer.Glyph().GetRuneIndex('M')
	mWidth := sizer.GlyphAdvance(font, buffer, size, mIndex)
	
	numLines := int(float64(self.lastHeight)/(lineHeight.ToFloat64()*HeightFactor) - 1)
	numChars := int(float64(self.lastWidth)/(mWidth.ToFloat64()*WidthFactor) - 1)

	// expand or collapse to the correct number of lines
	if len(self.backLines) > numLines {
		self.backLines = self.backLines[0 : numLines]
	}
	for len(self.backLines) < numLines {
		self.backLines = append(self.backLines, make([]rune, 0, numChars))
	}

	// expand or collapse to the correct number of runes per line
	for i := 0; i < len(self.backLines); i++ {
		if len(self.backLines[i]) == numChars { continue }
		if len(self.backLines[i]) > numChars {
			self.backLines[i] = self.backLines[i][0 : numChars]
		}
		for len(self.backLines[i]) < numChars {
			codePoint := runePool[rand.Intn(len(runePool))]
			self.backLines[i] = append(self.backLines[i], codePoint)
		}
	}
}

func (self *Game) Update() error {
	// background debug toggle
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		self.backOnly = !self.backOnly
	}

	// glitch effect toggle (so bad that it starts disabled)
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		self.glitchesEnabled = !self.glitchesEnabled
	}

	// update background text by randomly changing
	// some characters from time to time
	self.ensureBackLinesContent()
	randMaxOpen := len(runePool)
	for _, line := range self.backLines {
		for i, _ := range line {
			if rand.Float64() < 0.0625 { // change runes arbitrarily
				line[i] = runePool[rand.Intn(randMaxOpen)]
			}
		}
	}

	// update glitch rects
	for i, rect := range self.glitchRects {
		self.glitchRects[i] = self.refreshGlitchRect(rect)
	}
	
	return nil
}

func (self *Game) refreshGlitchRect(rect image.Rectangle) image.Rectangle {
	if rect.Empty() {
		if rand.Float64() < 0.02 {
			ox := rand.Intn(self.lastWidth)
			oy := rand.Intn(self.lastHeight)
			var w, h int
			if rand.Float64() < 0.5 {
				w = rand.Intn(self.lastWidth/2) + self.lastWidth/8
				h = rand.Intn(self.lastWidth/8) + self.lastWidth/64
			} else {
				w = rand.Intn(self.lastHeight/8) + self.lastHeight/64
				h = rand.Intn(self.lastHeight/2) + self.lastHeight/8
			}
			return image.Rect(ox, oy, ox + w, oy + h)
		}
	} else if rand.Float64() < 0.14 {
		return image.Rect(0, 0, 0, 0)
	}
	return rect
}

func (self *Game) Draw(canvas *ebiten.Image) {
	// get canvas basic metrics
	bounds := canvas.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	
	// dark background
	canvas.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw background text
	// ... the main idea is to draw line by line while positioning
	//     the glyphs manually, more or less centered.
	buffer := self.backRenderer.GetBuffer()
	font   := self.backRenderer.GetFont()
	size   := self.backRenderer.Fract().GetScaledSize()
	sizer  := self.backRenderer.GetSizer()
	lineHeight := sizer.LineHeight(font, buffer, size)
	ascent     := sizer.Ascent(font, buffer, size)
	yShift := ascent - lineHeight/2

	numLines := len(self.backLines)
	var numChars int
	if numLines > 0 { numChars = len(self.backLines[0]) }
	xAdvance := fract.FromFloat64(float64(w)/float64(numChars + 1))
	yAdvance := fract.FromFloat64(float64(h)/float64(numLines + 1))
	var position fract.Point = fract.UnitsToPoint(xAdvance, yAdvance)
	for _, line := range self.backLines {
		for _, codePoint := range line {
			// get glyph index and find its centered position
			glyphIndex := self.backRenderer.Glyph().GetRuneIndex(codePoint)
			glyphAdvance := sizer.GlyphAdvance(font, buffer, size, glyphIndex)
			origin := position.AddUnits(-glyphAdvance/2, yShift)
			origin.X = origin.X.QuantizeUp(etxt.QtFull)
			origin.Y = origin.Y.QuantizeUp(etxt.QtFull)

			// load mask, draw it, advance position
			mask := self.backRenderer.Glyph().LoadMask(glyphIndex, origin)
			self.backRenderer.Glyph().DrawMask(canvas, mask, origin)
			position.X += xAdvance
		}
		position.X = xAdvance
		position.Y += yAdvance
	}

	// initialize or resize offscreen if necessary for foreground text
	if self.offscreen == nil || !self.offscreen.Bounds().Eq(bounds) {
		self.offscreen = ebiten.NewImage(w, h)
	}

	if !self.backOnly {
		// draw front text to offscreen image (actually, since the
		// blend mode is XOR, it's actually "cutting out" the text)
		self.offscreen.Fill(self.frontRenderer.GetColor())
		self.frontRenderer.Draw(self.offscreen, MainText, w/2, h/2) // blend mode set in main()

		// draw glitch rects
		if self.glitchesEnabled {
			for _, rect := range self.glitchRects {
				self.drawGlitchRect(rect)
			}
		}

		// draw offscreen over canvas
		canvas.DrawImage(self.offscreen, nil)
	}
}

func (self *Game) drawGlitchRect(rect image.Rectangle) {
	if !rect.Empty() {
		glitchSub := self.offscreen.SubImage(rect).(*ebiten.Image)
		glitchSub.Fill(color.RGBA{0, 0, 0, 0})
	}
}

func main() {
	// seed rng (unnecessary on go1.20 and later)
	rand.Seed(time.Now().UnixNano())

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

	// for this example we will create two renderers instead
	// of just one. it could also be done with a single one,
	// but I personally think having two renderers makes life
	// slightly easier in this case.
	backRenderer := etxt.NewRenderer()
	backRenderer.Utils().SetCache8MiB()
	backRenderer.SetSize(16)
	backRenderer.SetFont(sfntFont)
	backRenderer.SetAlign(etxt.Baseline | etxt.Left)
	backRenderer.SetColor(color.RGBA{0, 255, 0, 255})

	frontRenderer := etxt.NewRenderer()
	frontRenderer.Utils().SetCache8MiB() // share cache (getting tight, though)
	frontRenderer.SetSize(MainFontSize)
	frontRenderer.SetFont(sfntFont)
	frontRenderer.SetAlign(etxt.Center)
	frontRenderer.SetColor(color.RGBA{0, 244, 0, 244}) // [1]
	frontRenderer.SetBlendMode(ebiten.BlendXor) // **the critical part**
	// [1] I generally like the textures created by slight translucency,
	//     but you can also use 255 for the solid color (or 0 to see the
	//     background weirdness in all its glory).

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/cutout")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game{
		backRenderer: backRenderer,
		frontRenderer: frontRenderer,
	})
	if err != nil { log.Fatal(err) }
}
