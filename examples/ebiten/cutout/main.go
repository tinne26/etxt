//go:build !gtxt

package main

import "os"
import "log"
import "fmt"
import "time"
import "math"
import "math/rand"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt"

// NOTICE: this program looks very different with thick and slim
//         fonts. Artsy with the slim ones, nerdy with the thick
//         ones. Try out different fonts!
const MainText     = "COMPLETE\nSYSTEM\nFAILURE"
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
}
func (self *Game) Layout(w int, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	return int(math.Ceil(float64(w)*scale)), int(math.Ceil(float64(h)*scale))
}
func (self *Game) Update() error {
	// update background text
	randMaxOpen := len(runePool)
	for _, line := range self.backLines {
		for i, _ := range line {
			if rand.Int63n(1024) < 64 { // change runes arbitrarily
				line[i] = runePool[rand.Intn(randMaxOpen)]
			}
		}
	}
	
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw background text
	// ... the main idea is to draw line by line while positioning
	//     the glyphs manually, more or less centered.
	scale := ebiten.DeviceScaleFactor()
	self.backRenderer.SetTarget(screen)
	for i, line := range self.backLines {
		y := fixed.Int26_6(int(scale*float64((22 + i*16)*64)))
		x := fixed.Int26_6(int(scale*12*64))
		self.backRenderer.Traverse(string(line), fixed.P(0, 0),
			func (dot fixed.Point26_6, _ rune, index etxt.GlyphIndex) {
				mask := self.backRenderer.LoadGlyphMask(index, dot)
				glyphWidth, glyphHeight := mask.Size()
				dot.X = x - fixed.Int26_6(glyphWidth << 5)
				dot.Y = y - fixed.Int26_6(glyphHeight << 5)
				self.backRenderer.DefaultDrawFunc(dot, mask, index)
				x += fixed.Int26_6(scale*16*64)
			})
	}

	// draw front text to offscreen image
	w, h := screen.Size()
	if self.offscreen == nil || sizeMismatch(self.offscreen, w, h) {
		self.offscreen = ebiten.NewImage(w, h)
		self.frontRenderer.SetTarget(self.offscreen)
	}
	self.offscreen.Fill(self.frontRenderer.GetColor())
	self.frontRenderer.Draw(MainText, w/2, h/2) // mix mode was set in main()

	// draw offscreen over screen
	screen.DrawImage(self.offscreen, nil)
}

func sizeMismatch(img *ebiten.Image, expectedWidth, expectedHeight int) bool {
	width, height := img.Size()
	return (width != expectedWidth) || (height != expectedHeight)
}

func main() {
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
	renderer.SetSizePx(int(16*ebiten.DeviceScaleFactor()))
	renderer.SetFont(font)
	renderer.SetAlign(etxt.Baseline, etxt.Left)
	renderer.SetColor(color.RGBA{0, 255, 0, 255})

	// mmmm... actually, two renderers will make life easier here
	frontRend := etxt.NewRenderer(renderer.GetRasterizer()) // share rasterizer, no problem
	frontRend.SetCacheHandler(cache.NewHandler()) // share cache, no problem
	frontRend.SetSizePx(int(MainFontSize*ebiten.DeviceScaleFactor()))
	frontRend.SetFont(font)
	frontRend.SetAlign(etxt.YCenter, etxt.XCenter)
	frontRend.SetColor(color.RGBA{0, 255, 0, 244}) // [1]
	frontRend.SetMixMode(ebiten.CompositeModeXor) // **the critical part**
	// [1] I generally like the textures created by slight translucency,
	//     but you can also use 255 for the solid color (or 0 to see the
	//     background weirdness in all its glory).

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/cutout")
	ebiten.SetWindowSize(640, 480)
	backLines := make([][]rune, 480/16)
	for i, _ := range backLines {
		runes := make([]rune, 640/16)
		for i, _ := range runes { runes[i] = '0' }
		backLines[i] = runes
	}
	err = ebiten.RunGame(&Game { renderer, frontRend, backLines, nil })
	if err != nil { log.Fatal(err) }
}
