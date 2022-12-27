package main

import "os"
import "log"
import "fmt"
import "time"
import "math"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/esizer"

// NOTE: to honor the beauty of fonts, I decided to make this example
//       resizable and fullscreen-able (press F, but release fast or
//       it will re-trigger). You can navigate glyphs with the up/down
//       arrows. You can also press H to switch on/off the current glyph
//       info. Enjoy your fonts!

const GlyphSize    = 64   // controls the glyph size
const GlyphSpacing = 1.24 // controls the space around/between glyphs

type Game struct {
	txtRenderer *etxt.Renderer
	numGlyphs int
	position int
	spacing int
	lastPressTime time.Time
	glyphs []etxt.GlyphIndex

	screenWidth int
	screenHeight int
	hint bool
	hintRenderer *etxt.Renderer
	bottomFade *ebiten.Image
}

func NewGame(renderer *etxt.Renderer, font *etxt.Font, noHint bool) *Game {
	numGlyphs := font.NumGlyphs()
	fixedSizer := &esizer.FixedSizer{}
	renderer.SetSizer(fixedSizer)

	var hintRenderer *etxt.Renderer
	if !noHint {
		hintRenderer = etxt.NewStdRenderer()
		hintRenderer.SetFont(font)
		hintRenderer.SetSizePx(int(14*ebiten.DeviceScaleFactor()))
		hintRenderer.SetHorzAlign(etxt.Right)
		cache := etxt.NewDefaultCache(1024*1024) // 1MB cache
		hintRenderer.SetCacheHandler(cache.NewHandler())
		hintRenderer.SetColor(color.RGBA{92, 92, 92, 255})
	}

	game := &Game {
		txtRenderer: renderer,
		numGlyphs: numGlyphs,
		position: 0,
		spacing: 0,
		lastPressTime: time.Now(),
		screenWidth: 640,
		screenHeight: 480,
		hint: true,
		hintRenderer: hintRenderer,
	}
	game.refreshScreenProperties()
	return game
}

func (self *Game) refreshScreenProperties() {
	size := self.txtRenderer.GetSizePxFract()
	spacing := int((float64(size)*GlyphSpacing)/64.0)
	sizer := self.txtRenderer.GetSizer().(*esizer.FixedSizer)
	sizer.SetAdvance(spacing)
	glyphsPerLine := self.screenWidth/spacing
	if glyphsPerLine != len(self.glyphs) {
		self.glyphs = make([]etxt.GlyphIndex, glyphsPerLine)
	}
	self.spacing = spacing
	self.bottomFade = ebiten.NewImage(self.screenWidth, 16)
	self.bottomFade.Fill(color.RGBA{0, 0, 0, 64})
}

func (self *Game) Layout(w int, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	sw, sh := int(math.Ceil(float64(w)*scale)), int(math.Ceil(float64(h)*scale))
	if sw != self.screenWidth || sh != self.screenHeight {
		self.screenWidth  = sw
		self.screenHeight = sh
		self.refreshScreenProperties()
	}
	return sw, sh
}

func (self *Game) Update() error {
	now := time.Now()
	if now.Sub(self.lastPressTime) < time.Millisecond*250 {
		return nil
	}

	if ebiten.IsKeyPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		self.lastPressTime = now
		return nil
	}

	if ebiten.IsKeyPressed(ebiten.KeyH) {
		self.hint = !self.hint
		self.lastPressTime = now
		return nil
	}

	up   := ebiten.IsKeyPressed(ebiten.KeyUp)
	down := ebiten.IsKeyPressed(ebiten.KeyDown)
	glyphsPerLine := len(self.glyphs)
	line := self.position/glyphsPerLine
	maxLine := (self.numGlyphs + glyphsPerLine - 1)/glyphsPerLine
	maxLine -= (self.screenHeight/self.spacing)
	if self.numGlyphs > glyphsPerLine { maxLine += 1 }
	if maxLine < 0 { maxLine = 0 }

	if self.position > 0 && up {
		self.position -= glyphsPerLine
		if self.position < 0 { self.position = 0 }
		self.lastPressTime = now
	} else if line < maxLine && down {
		self.position += glyphsPerLine
		if self.position >= self.numGlyphs {
			self.position = self.numGlyphs - 1
		}
		self.lastPressTime = now
	}
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw text
	self.txtRenderer.SetTarget(screen)
	currentPos := self.position
	sp := self.spacing // I'm gonna use this to death
	lineWidth  := sp*len(self.glyphs) - sp/3
	leftMargin := (self.screenWidth - lineWidth)/2
	for i := 0; i < (self.screenHeight + sp - 1)/sp; i++ {
		// set the glyph indices to draw
		lastPos := currentPos + len(self.glyphs)
		if lastPos >= self.numGlyphs { lastPos = self.numGlyphs }
		if lastPos <= currentPos { continue }
		for i := 0; currentPos + i < lastPos; i++ {
			self.glyphs[i] = etxt.GlyphIndex(currentPos + i)
		}

		// draw the glyphs
		glyphs := self.glyphs[0 : lastPos - currentPos]
		x, y := leftMargin, (sp*3)/4 + i*sp
		self.txtRenderer.TraverseGlyphs(glyphs, fixed.P(x, y),
			func (dot fixed.Point26_6, index etxt.GlyphIndex) {
				mask := self.txtRenderer.LoadGlyphMask(index, dot)
				self.txtRenderer.DefaultDrawFunc(dot, mask, index)
			})

		// advance to next glyphs
		currentPos = lastPos
	}

	// make the bottom fade out
	for i := 0; i < 16; i++ {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(0, float64(self.screenHeight - 16 + i))
		screen.DrawImage(self.bottomFade, opts)
	}

	// draw current glyph index and total
	if self.hint && self.hintRenderer != nil {
		self.hintRenderer.SetTarget(screen)
		text := fmt.Sprintf("%d of %d glyphs", self.position + 1, self.numGlyphs)
		self.hintRenderer.Draw(text, self.screenWidth - 4, self.screenHeight - 6)
		self.hintRenderer.SetTarget(nil)
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
	font, fontName, err := etxt.ParseFontFrom(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// create cache
	cache := etxt.NewDefaultCache(10*1024*1024) // 10MB
	// **IMPORTANT CACHE INFO**
	// In almost every example we have been setting caches to 1GB,
	// mostly because we weren't expecting to fill them anywhere near
	// that. In this example, instead, we are setting the cache to a
	// more reasonable value, because otherwise the cache could really
	// fill up a lot for some fonts.
	//
	// This example is almost a textbook situation for a cache: glyphs
	// only appear once, and if they are on the screen they will be
	// heavily reused, but once we scroll past them, they aren't likely
	// to come up again. And... if they come up again, it is because we
	// saw them recently and scrolled back to them. So least recently
	// used policies are a perfect fit for this. Technically, the default
	// cache uses random sampling for eviction, not perfect LRU, but
	// the results should be quite decent anyway.

	// create and configure renderer
	renderer := etxt.NewStdRenderer()
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(int(ebiten.DeviceScaleFactor()*GlyphSize))
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.Left)

	// determine if we have the right glyphs to show hint text
	const alphabet = " ofglyphs0123456789"
	missing, err := etxt.GetMissingRunes(font, alphabet)
	if err != nil { log.Fatal(err) }

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/all_glyphs")
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowSize(640, 480)
	ebiten.SetCursorMode(ebiten.CursorModeHidden) // doing this right, boys...
	err = ebiten.RunGame(NewGame(renderer, font, len(missing) > 0))
	if err != nil { log.Fatal(err) }
}
