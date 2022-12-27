package main

import "os"
import "log"
import "fmt"
import "math"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/emask"
import "github.com/tinne26/etxt/esizer"

const MainText = "The lazy programmer jumps\nover the brown codebase."

type Game struct {
	fauxRenderer *etxt.Renderer
	helpRenderer *etxt.Renderer
	italicAngle float64 // native italic angle for the font

	skewFactor float64 // [-1, 1]
	extraWidth float64 // [0, 10]
	sinceLastKey int
	quantized bool

	usingCustomSizer bool
	fauxSizer esizer.Sizer
	defaultSizer esizer.Sizer
}

func (self *Game) Layout(w int, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	return int(math.Ceil(float64(w)*scale)), int(math.Ceil(float64(h)*scale))
}
func (self *Game) Update() error {
	self.sinceLastKey += 1

	// left/right to modify skew (oblique)
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		if self.applyArrowSkewChange(+1) { return nil }
	} else if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		if self.applyArrowSkewChange(-1) { return nil }
	}

	// up/down to modify width (bold)
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if self.applyArrowBoldChange(+1) { return nil}
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		if self.applyArrowBoldChange(-1) { return nil}
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		if self.sinceLastKey > 20 {
			self.sinceLastKey = 0
			if self.usingCustomSizer {
				self.fauxRenderer.SetSizer(self.defaultSizer)
			} else {
				self.fauxRenderer.SetSizer(self.fauxSizer)
			}
			self.usingCustomSizer = !self.usingCustomSizer
		}
	}

	// unitalicize
	hasAngle := (self.italicAngle != 0) && (!math.IsNaN(self.italicAngle))
	if hasAngle && ebiten.IsKeyPressed(ebiten.KeyU) {
		if self.sinceLastKey > 20 {
			// NOTE: I've tried with a few google fonts... and the angles are
			//       not very reliable or accurate. I don't know what the heck
			//       do they do to measure angles. Maybe they don't even measure
			//       them, but if they are wrong by only 2 degrees consider
			//       yourself lucky...
			newSkew := self.italicAngle/45.0
			if newSkew != self.skewFactor {
				self.skewFactor = newSkew
				self.refreshSkew()
				self.sinceLastKey = 0
			}
		}
	}

	// reset key (resets bold and oblique)
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		if self.sinceLastKey > 20 {
			self.sinceLastKey = 0
			fauxRast := self.fauxRenderer.GetRasterizer()
			fauxRast.(*emask.FauxRasterizer).SetSkewFactor(0)
			fauxRast.(*emask.FauxRasterizer).SetExtraWidth(0)
			self.extraWidth = 0
			self.skewFactor = 0
			return nil
		}
	}

	// quantization switch
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		if self.sinceLastKey > 20 {
			self.sinceLastKey = 0
			if self.quantized {
				self.fauxRenderer.SetQuantizerStep(1, 64)
			} else {
				self.fauxRenderer.SetQuantizerStep(64, 64)
			}
			self.quantized = !self.quantized
			return nil
		}
	}

	return nil
}

// logic to modify the skewFactor (for oblique text)
func (self *Game) applyArrowSkewChange(sign int) bool {
	if self.sinceLastKey < 10 { return false }

	var skewAbsChange float64
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		skewAbsChange = 0.01
	} else {
		skewAbsChange = 0.03
	}

	var newSkew float64
	if sign >= 0 {
		newSkew = self.skewFactor + skewAbsChange
		if newSkew > 1.0 { newSkew = 1.0 }
	} else {
		newSkew = self.skewFactor - skewAbsChange
		if newSkew < -1.0 { newSkew = -1.0 }
	}

	if newSkew == self.skewFactor { return false }
	self.skewFactor = newSkew
	self.refreshSkew()
	self.sinceLastKey = 0
	return true
}

// logic to modify the extraWidth (for faux-bold text)
func (self *Game) applyArrowBoldChange(sign int) bool {
	if self.sinceLastKey < 20 { return false }

	var boldAbsChange float64
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		boldAbsChange = 0.2
	} else {
		boldAbsChange = 0.5
	}

	var newBold float64
	if sign >= 0 {
		newBold = self.extraWidth + boldAbsChange
		if newBold > 10.0 { newBold = 10.0 }
	} else {
		newBold = self.extraWidth - boldAbsChange
		if newBold <  0.0 { newBold =  0.0 }
	}

	if newBold == self.extraWidth { return false }
	self.extraWidth = newBold
	self.refreshBold()
	self.sinceLastKey = 0
	return true
}

// Updates the rasterizer's skew factor.
func (self *Game) refreshSkew() {
	fauxRast := self.fauxRenderer.GetRasterizer()
	fauxRast.(*emask.FauxRasterizer).SetSkewFactor(self.skewFactor)
}

// Updates the rasterizer's extraWidth.
func (self *Game) refreshBold() {
	fauxRast := self.fauxRenderer.GetRasterizer()
	fauxRast.(*emask.FauxRasterizer).SetExtraWidth(self.extraWidth)
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw text
	w, h := screen.Size()
	self.fauxRenderer.SetTarget(screen)
	self.fauxRenderer.Draw(MainText, w/2, h/3)

	// draw helper info
	skewInfo := fmt.Sprintf("skew %.2f (%.2f degrees) [right/left]", self.skewFactor, -self.skewFactor*45)
	y := h - h/3 - int(float64(self.helpRenderer.GetLineAdvance().Ceil())*3)
	self.helpRenderer.SetTarget(screen)

	self.helpRenderer.Draw(skewInfo, w/2, y)
	y += self.helpRenderer.GetLineAdvance().Ceil()
	if math.IsNaN(self.italicAngle) {
		self.helpRenderer.Draw("original italic angle unknown", w/2, y)
	} else {
		var info string
		if self.italicAngle == 0 {
			info = fmt.Sprintf("original italic angle %.2f degrees", self.italicAngle)
		} else {
			info = fmt.Sprintf("orig. it. angle %.2f degrees [U unitalicize]", self.italicAngle)
		}
		self.helpRenderer.Draw(info, w/2, y)
	}
	y += self.helpRenderer.GetLineAdvance().Ceil()
	boldInfo := fmt.Sprintf("bold +%.1fpx [up/down]", self.extraWidth)
	self.helpRenderer.Draw(boldInfo, w/2, y)
	y += self.helpRenderer.GetLineAdvance().Ceil()
	if self.quantized {
		self.helpRenderer.Draw("quantization ON [press Q]", w/2, y)
	} else {
		self.helpRenderer.Draw("quantization OFF [press Q]", w/2, y)
	}
	y += self.helpRenderer.GetLineAdvance().Ceil()
	if self.usingCustomSizer {
		self.helpRenderer.Draw("faux sizer ON [press S]", w/2, y)
	} else {
		self.helpRenderer.Draw("faux sizer OFF [press S]", w/2, y)
	}
	y += self.helpRenderer.GetLineAdvance().Ceil()
	self.helpRenderer.Draw("[press R to Reset]", w/2, y)
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
	cache := etxt.NewDefaultCache(1024*1024) // 1MB cache

	// create and configure renderer
	fauxRast := emask.FauxRasterizer{}
	renderer := etxt.NewRenderer(&fauxRast)
	defaultSizer := renderer.GetSizer()
	customSizer  := &esizer.AdvancePadSizer{}
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(int(36*ebiten.DeviceScaleFactor()))
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{255, 255, 255, 255})

	// link custom sizer to fauxRast
	fauxRast.SetAuxOnChangeFunc(func(*emask.FauxRasterizer) {
		const SpFactor = 0.5 // between 0.5 and 1.0 is ok
		customSizer.SetPaddingFloat(SpFactor*fauxRast.GetExtraWidth())
	})

	// create helper renderer for other text
	helpRend := etxt.NewStdRenderer()
	helpRend.SetCacheHandler(cache.NewHandler())
	helpRend.SetSizePx(int(16*ebiten.DeviceScaleFactor()))
	helpRend.SetQuantizerStep(1, 64)
	helpRend.SetFont(font)
	helpRend.SetAlign(etxt.YCenter, etxt.XCenter)
	helpRend.SetColor(color.RGBA{255, 255, 255, 150})

	// get original italic angle information
	postTable := font.PostTable()
	italicAngle := math.NaN()
	if postTable != nil { italicAngle = postTable.ItalicAngle }

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/faux_styles")
	ebiten.SetWindowSize(640, 480)
	err = ebiten.RunGame(&Game {
		fauxRenderer: renderer,
		helpRenderer: helpRend,
		quantized: true,
		fauxSizer: customSizer,
		defaultSizer: defaultSizer,
		italicAngle: italicAngle,
	})
	if err != nil { log.Fatal(err) }
}
