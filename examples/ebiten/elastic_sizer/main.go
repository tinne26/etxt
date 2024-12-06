package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/cache"
	"github.com/tinne26/etxt/font"
	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/etxt/sizer"
)

// This example showcases how to modify the text quantization level,
// and how this can be critical for smooth text animations when they
// involve movement. You can run the example like this:
//   go run github.com/tinne26/etxt/examples/ebiten/elastic_sizer@latest path/to/font.ttf
//
// This example also shows how to use a cache in slightly more advanced
// ways than most other examples. In particular, this example does
// pre-caching and creates the cache manually instead of relying on
// Renderer.Utils().SetCache8MiB().

// you can play around with these, but it can get out of hand quite easily
const SpringText = "Bouncy!"
const MainTextSize = 64
const InfoTextSize = 14
const MinExpansion = 0.34      // must be strictly below 1.0
const MaxExpansion = 4.0       // must be strictly above 1.0
const Timescaling = 0.8 / 40.0 // make the first factor smaller to slow down
const Bounciness = 25.0

type Game struct {
	text *etxt.Renderer

	// spring related variables
	restLength float64
	textLen    float64 // number of code points in SpringText - 1
	expansion  float64 // between MinExpansion - MaxExpansion
	inertia    float64
	holdX      int
	holding    bool
	qPressed   bool
}

func NewGame(renderer *etxt.Renderer) *Game {
	renderer.SetScale(ebiten.DeviceScaleFactor())
	renderer.SetSize(MainTextSize)
	textRect := renderer.Measure(SpringText)

	// caching example (not strictly necessary)
	precacheText(renderer, SpringText)
	renderer.SetSize(InfoTextSize)
	precacheText(renderer, "0123456789QOCFPSacdeghklnoqrtuy[]()")

	return &Game{
		text:       renderer,
		restLength: textRect.Width().ToFloat64(),
		textLen:    float64(len([]rune(SpringText))),
		expansion:  1.0,
		inertia:    0.0,
		holdX:      0,
		holding:    false,
		qPressed:   false,
	}
}

func (self *Game) Layout(winWidth int, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth := int(math.Ceil(float64(winWidth) * scale))
	canvasHeight := int(math.Ceil(float64(winHeight) * scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	// Logic for switching quantization on / off
	newQPressed := ebiten.IsKeyPressed(ebiten.KeyQ)
	if self.qPressed != newQPressed {
		if !self.qPressed {
			horzQuant, _ := self.text.Fract().GetQuantization()
			if horzQuant == etxt.QtFull {
				self.text.Fract().SetHorzQuantization(etxt.QtNone)
			} else {
				self.text.Fract().SetHorzQuantization(etxt.QtFull)
			}
		}
		self.qPressed = newQPressed
	}

	// Spring simulation logic. This part of the code doesn't have
	// anything to do with text rendering, so you can ignore it.
	// It's not like the spring simulation is even good (too linear).
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		// manual spring manipulation with the mouse
		if !self.holding {
			// just started to hold
			self.holding = true
			self.holdX, _ = ebiten.CursorPosition()
		} else {
			// continue holding and moving
			newHold, _ := ebiten.CursorPosition()
			diff := newHold - self.holdX
			self.holdX = newHold
			expansionChange := float64(diff) / self.restLength
			self.expansion += expansionChange
			if self.expansion < MinExpansion {
				self.expansion = MinExpansion
			}
			if self.expansion > MaxExpansion {
				self.expansion = MaxExpansion
			}
		}
	} else { // spring simulation
		self.holding = false
		var tension float64
		workingLength := (MaxExpansion - MinExpansion) * self.restLength
		if self.expansion < 1.0 {
			tension = ((1.0 - self.expansion) / (1.0 - MinExpansion)) * workingLength
		} else { // expansion >= 1.0
			tension = -((self.expansion - 1.0) / (MaxExpansion - 1.0)) * workingLength
		}

		// apply movement and update inertia
		movement := (self.inertia + tension) * Timescaling
		self.inertia += Bounciness * tension * Timescaling * ebiten.DeviceScaleFactor()
		self.expansion = self.expansion + (movement / self.restLength)

		// clamp expansion if it went outside range
		if self.expansion < MinExpansion {
			self.expansion = MinExpansion
			self.inertia = 0
		}
		if self.expansion > MaxExpansion {
			self.expansion = MaxExpansion
			self.inertia = 0
		}
	}

	return nil
}

// For the purposes of this example, the only key lines are
// the ones where we get the sizer and set its padding, but
// the rest should still be nice for general reference.
func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// get and adjust sizer (we could have stored it earlier too, but no need)
	sizer := self.text.GetSizer().(*sizer.PaddedKernSizer)
	letterPad := (self.expansion*self.restLength - self.restLength) / self.textLen
	sizer.SetPadding(fract.FromFloat64(letterPad))

	// get screen size
	screenBounds := screen.Bounds()
	sw, sh := screenBounds.Dx(), screenBounds.Dy()

	// draw text
	self.text.SetSize(MainTextSize)
	self.text.SetAlign(etxt.VertCenter | etxt.Left)
	self.text.SetColor(color.RGBA{255, 255, 255, 255})
	self.text.Draw(screen, SpringText, sw/16, sh/2)

	// draw fps and instructions text
	sizer.SetPadding(0)
	self.text.SetSize(InfoTextSize)
	self.text.SetColor(color.RGBA{128, 128, 128, 128})
	self.text.SetAlign(etxt.Baseline) // vertical

	// (fps on the right side)
	self.text.SetAlign(etxt.Right)
	self.text.Draw(screen, fmt.Sprintf("%.2f FPS", ebiten.ActualFPS()), sw-sh/32, sh-sh/32)

	// (quantization in the middle)
	self.text.SetAlign(etxt.HorzCenter)
	horzQuant, _ := self.text.Fract().GetQuantization()
	if horzQuant == etxt.QtFull {
		self.text.Draw(screen, "Quantization ON [Q]", sw/2, sh-sh/32)
	} else {
		self.text.Draw(screen, "Quantization OFF [Q]", sw/2, sh-sh/32)
	}

	// (instructions on the left side)
	self.text.SetAlign(etxt.Left)
	instructions := "Click and drag horizontally"
	if self.holding {
		instructions += " (holding)"
	}
	self.text.Draw(screen, instructions, sh/32, sh-sh/32)
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

	// create cache manually as we want it to be fairly big
	glyphCache := cache.NewDefaultCache(512 * 1024 * 1024) // 512MiB cache

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.SetCacheHandler(glyphCache.NewHandler())
	renderer.SetFont(sfntFont)
	renderer.SetSizer(&sizer.PaddedKernSizer{})
	renderer.Fract().SetHorzQuantization(etxt.QtNone) // *
	// * Disabling horizontal quantization is helpful here to get
	//   smoother results. But it also means we have to cache each
	//   glyph in 64 different positions! At big sizes this is not
	//   cheap. The program allows pressing [Q] to see the results
	//   with and without quantization.

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/elastic_sizer")
	ebiten.SetWindowSize(840, 360)
	err = ebiten.RunGame(NewGame(renderer))
	if err != nil {
		log.Fatal(err)
	}
}

// This code has been added mostly to provide an example of
// how to manually cache text at fractional px positions.
//
// Notice that the font, size, scale and quantization mode
// must be already properly set if we want the caching to be
// meaningful.
func precacheText(renderer *etxt.Renderer, text string) {
	for _, codePoint := range text {
		index := renderer.Glyph().GetRuneIndex(codePoint)
		renderer.Glyph().CacheIndex(index)
	}

	// print info about cache size
	cache := renderer.GetCacheHandler().(*cache.DefaultCacheHandler).Cache()
	peakSize := cache.PeakSize()
	mbSize := float64(peakSize) / (1024 * 1024)
	numEntries := cache.NumEntries()
	fmt.Printf("Cache size after pre-caching: %d entries, %d bytes (%.2fMB)\n", numEntries, peakSize, mbSize)
}
