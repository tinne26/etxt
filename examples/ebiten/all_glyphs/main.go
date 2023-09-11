package main

import "os"
import "log"
import "fmt"
import "math"
import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/inpututil"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"

// This example is more of an homage to beautiful typography than
// anything else. You can basically see all the glyphs of a font,
// navigating with up/down arrows and using F to go fullscreen.
// You can also press H to hide/show the glyph indices info.
//
// The only interesting part of the example is probably the fact
// that it uses a low-level drawing mechanism, getting info directly
// from the sizer, aligning glyphs manually and drawing them with
// Renderer.Glyph().DrawMask().
// 
// You can run it like this:
//   go run github.com/tinne26/etxt/examples/ebiten/all_glyphs@latest path/to/font.ttf
//
// Enjoy your fonts!

const GlyphSize    = 50   // controls the glyph size
const GlyphSpacing = 1.25 // controls the space around/between glyphs

type Game struct {
	text *etxt.Renderer
	numGlyphs int
	glyphIndex int
	glyphsPerLine int
	showHints bool
	canShowHints bool
	fadeShader *ebiten.Shader
}

// Simple shader for a bottom fade effect.
var shaderSrc []byte = []byte(`
package main
func Fragment(_ vec4, _ vec2, color vec4) vec4 {
	return vec4(0, 0, 0, ease(color.a))
}

func ease(t float) float { // ease out cubic
	return 1.0 - pow(1.0 - t, 3.0)
}
`)

func (self *Game) Layout(_, _ int) (int, int) { panic("use Ebitengine >=v2.5.0") }
func (self *Game) LayoutF(logicWinWidth, logicWinHeight float64) (float64, float64) {
	self.glyphsPerLine = int(logicWinWidth/(GlyphSize*GlyphSpacing))
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale)
	canvasWidth  := math.Ceil(logicWinWidth*scale)
	canvasHeight := math.Ceil(logicWinHeight*scale)
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	// fullscreen toggle
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		return nil
	}

	// hints toggle
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		self.showHints = !self.showHints
		return nil
	}

	// helper function
	var repeatTrigger = func(key ebiten.Key) bool {
		ticks := inpututil.KeyPressDuration(key)
		return ticks == 1 || (ticks > 14 && (ticks - 14) % 9 == 0)
	}	

	// move to previous line
	if self.glyphIndex > 0 && repeatTrigger(ebiten.KeyUp) {
		self.glyphIndex -= self.glyphsPerLine
		if self.glyphIndex < 0 { self.glyphIndex = 0 }
		return nil
	}

	// move to next line
	nextLineStart := self.glyphIndex + self.glyphsPerLine
	if nextLineStart < self.numGlyphs && repeatTrigger(ebiten.KeyDown) {
		self.glyphIndex += self.glyphsPerLine
		if self.glyphIndex >= self.numGlyphs {
			self.glyphIndex = self.numGlyphs - 1
		}
	}

	return nil
}

func (self *Game) Draw(canvas *ebiten.Image) {
	// dark background
	canvas.Fill(color.RGBA{ 0, 0, 0, 255 })

	// get canvas dimensions
	bounds := canvas.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// get some metrics
	scale := ebiten.DeviceScaleFactor()
	scaledGlyphSize := GlyphSize*scale
	xAdvance := fract.FromFloat64(GlyphSpacing*scaledGlyphSize)
	lineWidth := fract.FromInt(self.glyphsPerLine).Mul(xAdvance)
	xStart := (fract.FromInt(w) - lineWidth)/2

	buffer := self.text.GetBuffer()
	font   := self.text.GetFont()
	size   := self.text.Fract().GetScaledSize()
	sizer  := self.text.GetSizer()
	lineHeight := sizer.LineHeight(font, buffer, size).ToFloat64()
	ascent     := sizer.Ascent(font, buffer, size)

	// draw glyphs
	vertPad := fract.FromFloat64(12*scale)
	var position fract.Point = fract.UnitsToPoint(xStart, vertPad) // (top-left origin, not baseline)
	var index int = self.glyphIndex
	linesDrawn := 0
	for index < self.numGlyphs {
		for i := 0; i < self.glyphsPerLine; i++ {
			// get glyph advance, determine baseline drawing point, draw mask
			glyphIndex := sfnt.GlyphIndex(index)
			glyphAdvance := sizer.GlyphAdvance(font, buffer, size, glyphIndex)
			origin := position.AddUnits((xAdvance - glyphAdvance)/2, ascent)
			origin.X = origin.X.QuantizeUp(etxt.QtFull)
			mask := self.text.Glyph().LoadMask(glyphIndex, origin)
			self.text.Glyph().DrawMask(canvas, mask, origin)

			// increase index, advance position
			index += 1
			if index >= self.numGlyphs { break }
			position.X += xAdvance
		}

		linesDrawn += 1
		position.X = xStart
		position.Y += fract.FromFloat64(lineHeight*GlyphSpacing)
		position.Y = position.Y.QuantizeUp(etxt.QtFull)
		if position.Y.ToIntCeil() > h { break }
	}

	// draw bottom fade
	fh := int(GlyphSize*scale) // fade height
	canvas.DrawTrianglesShader(
		[]ebiten.Vertex{
			ebiten.Vertex{ DstX: 0, DstY: float32(h - fh), SrcX: 0, SrcY: 0, ColorA: 0 },
			ebiten.Vertex{ DstX: float32(w), DstY: float32(h - fh), SrcX: 0, SrcY: 0, ColorA: 0 },
			ebiten.Vertex{ DstX: 0, DstY: float32(h), SrcX: 0, SrcY: 0, ColorA: 1.0 },
			ebiten.Vertex{ DstX: float32(w), DstY: float32(h), SrcX: 0, SrcY: 0, ColorA: 1.0 },
		}, []uint16{0, 1, 2, 1, 3, 2}, self.fadeShader, nil,
	)

	// draw current glyph index and total
	if self.canShowHints && self.showHints {
		self.text.Utils().AssertMaxStoredStates(0)
		self.text.Utils().StoreState()
		self.text.SetAlign(etxt.Baseline | etxt.Right)
		self.text.SetColor(color.RGBA{92, 92, 92, 255})
		self.text.SetSize(14)
		text := fmt.Sprintf("%d..%d of %d glyphs", self.glyphIndex + 1, index, self.numGlyphs)
		self.text.Draw(canvas, text, w - int(8*scale), h - int(8*scale))
		self.text.Utils().RestoreState()
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
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB() // *
	renderer.SetSize(GlyphSize)
	renderer.SetFont(sfntFont)
	
	// * (random educational cache note)
	// This example is almost a textbook situation for a cache: glyphs
	// only appear once, and if they are on the screen they will be
	// heavily reused, but once we scroll past them, they aren't likely
	// to come up again. And... if they come up again, it is because we
	// saw them recently and scrolled back to them. So least recently
	// used policies are a perfect fit for this. Technically, the default
	// cache uses random sampling for eviction, not perfect LRU, but
	// the results should be quite decent anyway.

	// create the game struct
	shader, err := ebiten.NewShader(shaderSrc)
	if err != nil { log.Fatal(err) }
	missing, err := font.IsMissingRunes(sfntFont, " ofglyphs0123456789")
	if err != nil { log.Fatal(err) }
	game := &Game {
		text: renderer,
		numGlyphs: sfntFont.NumGlyphs(),
		glyphIndex: 0,
		showHints: !missing,
		canShowHints: !missing,
		fadeShader: shader,
	}

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/all_glyphs")
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowSize(640, 480)
	ebiten.SetCursorMode(ebiten.CursorModeHidden) // doing this right, boys...
	err = ebiten.RunGame(game)
	if err != nil { log.Fatal(err) }
}
