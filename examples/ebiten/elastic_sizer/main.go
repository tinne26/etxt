package main

import "os"
import "log"
import "fmt"
import "image/color"

import "golang.org/x/image/math/fixed"
import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/esizer"
import "github.com/tinne26/etxt/ecache"

// you can play around with these, but it can get out of hand quite easily
const SpringText   = "Bouncy!"
const MinExpansion = 0.34 // must be strictly below 1.0
const MaxExpansion = 4.0  // must be strictly above 1.0
const Timescaling  = 0.8/40.0 // make the first factor smaller to slow down
const Bounciness   = 25.0

type Game struct {
	txtRenderer *etxt.Renderer

	// spring related variables
	restLength float64
	textLen float64 // number of code points in SpringText - 1
	expansion float64 // between MinExpansion - MaxExpansion
	inertia float64
	holdX int
	holding bool
}

func NewGame(renderer *etxt.Renderer) *Game {
	textRect := renderer.SelectionRect(SpringText)
	precacheText(renderer) // not necessary, but a useful example
	return &Game {
		txtRenderer: renderer,
		restLength: float64(textRect.Width)/64,
		textLen: float64(len([]rune(SpringText))),
		expansion: 1.0,
		inertia: 0.0,
		holdX: 0,
		holding: false,
	}
}

func (self *Game) Layout(w int, h int) (int, int) { return w, h }
func (self *Game) Update() error {
	// All this code in Update() doesn't have much to do with text rendering
	// or anything, it's the spring simulation and related logic. It's the
	// most complex part of the program, but you may ignore it completely,
	// as the spring simulation is not even good (too linear).
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
			expansionChange := float64(diff)/self.restLength
			self.expansion += expansionChange
			if self.expansion < MinExpansion { self.expansion = MinExpansion }
			if self.expansion > MaxExpansion { self.expansion = MaxExpansion }
		}
	} else { // spring simulation
		self.holding = false
		var tension float64
		workingLength := (MaxExpansion - MinExpansion)*self.restLength
		if self.expansion < 1.0 {
			tension = ((1.0 - self.expansion)/(1.0 - MinExpansion))*workingLength
		} else { // expansion >= 1.0
			tension = -((self.expansion - 1.0)/(MaxExpansion - 1.0))*workingLength
		}

		// apply movement and update inertia
		movement := (self.inertia + tension)*Timescaling
		self.inertia += Bounciness*tension*Timescaling
		self.expansion = self.expansion + (movement/self.restLength)

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

// For the purposes of this example, the only key line is
// "sizer.SetHorzPaddingFloat"... but whatever, the others
// may be interesting too as general usage examples.
func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// get and adjust sizer (we could have stored it earlier too)
	sizer := self.txtRenderer.GetSizer().(*esizer.HorzPaddingSizer)
	sizer.SetHorzPaddingFloat((self.expansion*self.restLength - self.restLength)/self.textLen)

	// get some values that we will use later
	w, h := screen.Size()
	preVertAlign, preHorzAlign := self.txtRenderer.GetAlign()
	startX := 16
	if preHorzAlign == etxt.XCenter { startX = w/2 }

	// draw text
	self.txtRenderer.SetTarget(screen)
	self.txtRenderer.Draw(SpringText, startX, h/2)

	// draw fps and instructions text
	sizer.SetHorzPadding(0)
	preSize := self.txtRenderer.GetSizePxFract()

	self.txtRenderer.SetColor(color.RGBA{255, 255, 255, 128})
	self.txtRenderer.SetAlign(etxt.Baseline, etxt.Right)
	self.txtRenderer.SetSizePx(14)
	self.txtRenderer.Draw(fmt.Sprintf("%.2f FPS", ebiten.CurrentFPS()), w - 8, h - 8)
	self.txtRenderer.SetHorzAlign(etxt.Left)
	txt := "click and drag horizontally to interact"
	if self.holding { txt += " (holding)"}
	self.txtRenderer.Draw(txt, 8, h - 8)

	// restore renderer state after fps/instructions
	self.txtRenderer.SetColor(color.RGBA{255, 255, 255, 255})
	self.txtRenderer.SetAlign(preVertAlign, preHorzAlign)
	self.txtRenderer.SetSizePxFract(preSize)
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
	cache := etxt.NewDefaultCache(1024*1024*1024) // 1GB cache

	// create and configure renderer
	renderer := etxt.NewStdRenderer()
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(64)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.Left) // you can try etxt.XCenter too
	renderer.SetSizer(&esizer.HorzPaddingSizer{})
	renderer.SetQuantizerStep(1, 64) // *
	// * Disabling horizontal quantization is helpful here to get
	//   smoother results. But it also means we have to cache each
	//   glyph in 64 different positions! At big sizes this is not
	//   cheap. You can try commenting it out to see the difference.
	//   You may also use bigger step values and see how the animation
	//   becomes more or less smooth.

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/elastic")
	ebiten.SetWindowSize(840, 360)
	err = ebiten.RunGame(NewGame(renderer))
	if err != nil { log.Fatal(err) }
}

// This code has been added mostly to provide an example of how to manually
// cache text at fractional px positions. It might serve as a nice example of
// fixed.Int26_6 manipulation.
//
// More effective caching mechanisms may be added to etxt in the future,
// but this is a good example of how to do it by hand if required.
func precacheText(renderer *etxt.Renderer) {
	tmpTarget := ebiten.NewImage(1, 1)
	renderer.SetTarget(tmpTarget)
	for i := 0; i < 64; i++ {
		renderer.DrawFract(SpringText, fixed.Int26_6(i), 0)
	}
	renderer.SetTarget(nil)
	tmpTarget.Dispose()

	// print info about cache size
	cacheHandler := renderer.GetCacheHandler().(*ecache.DefaultCacheHandler)
	peakSize := cacheHandler.PeakCacheSize()
	mbSize := float64(peakSize)/1024/1024
	fmt.Printf("Cache size after pre-caching: %d bytes (%.2fMB)\n", peakSize, mbSize)
}
