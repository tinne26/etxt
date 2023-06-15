package main

import "os"
import "log"
import "fmt"
import "math"
import "strings"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/sizer"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"

const MainText = "The lazy programmer jumps\nover the brown codebase."

type Game struct {
	fauxRenderer *etxt.Renderer
	helpRenderer *etxt.Renderer
	italicAngle float64 // native italic angle for the font

	skewFactor float32 // [-1, 1]
	extraWidth float32 // [0, 10]
	sinceLastKey int
	quantized bool

	usingCustomSizer bool
	fauxSizer sizer.Sizer
	defaultSizer sizer.Sizer
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.fauxRenderer.SetScale(scale) // relevant for HiDPI
	self.helpRenderer.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}
func (self *Game) Update() error {
	// update counter to prevent excessive key repeat triggering
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

	// skip toggles and options if sinceLastKey is too recent
	if self.sinceLastKey <= 20 { return nil }

	// sizer switch
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		if self.usingCustomSizer {
			self.fauxRenderer.SetSizer(self.defaultSizer)
		} else {
			self.fauxRenderer.SetSizer(self.fauxSizer)
		}
		self.usingCustomSizer = !self.usingCustomSizer
		self.sinceLastKey = 0
		return nil
	}

	// quantization switch
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		if self.quantized {
			self.fauxRenderer.Fract().SetHorzQuantization(etxt.QtNone)
		} else {
			self.fauxRenderer.Fract().SetHorzQuantization(etxt.QtFull)
		}
		self.quantized = !self.quantized
		self.sinceLastKey = 0
		return nil
	}

	// unitalicize
	hasAngle := (self.italicAngle != 0) && (!math.IsNaN(self.italicAngle))
	if hasAngle && ebiten.IsKeyPressed(ebiten.KeyU) {
		// NOTE: I've tried with a few google fonts... and the angles are
		//       not very reliable or accurate. I don't know what the heck
		//       do they do to measure angles. Maybe they don't even measure
		//       them, but if they are wrong by only 2 degrees consider
		//       yourself lucky...
		newSkew := float32(self.italicAngle/45.0)
		if newSkew != self.skewFactor {
			self.skewFactor = newSkew
			self.refreshSkew()
			self.sinceLastKey = 0
			return nil
		}
	}

	// reset key (resets bold and oblique)
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		fauxRast := self.fauxRenderer.Glyph().GetRasterizer()
		fauxRast.(*mask.FauxRasterizer).SetSkewFactor(0)
		fauxRast.(*mask.FauxRasterizer).SetExtraWidth(0)
		self.extraWidth = 0
		self.skewFactor = 0
		self.sinceLastKey = 0
		return nil
	}

	return nil
}

// logic to modify the skewFactor (for oblique text)
func (self *Game) applyArrowSkewChange(sign int) bool {
	if self.sinceLastKey < 10 { return false }

	var skewAbsChange float32
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		skewAbsChange = 0.01
	} else {
		skewAbsChange = 0.03
	}

	var newSkew float32
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

	var boldAbsChange float32
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		boldAbsChange = 0.2
	} else {
		boldAbsChange = 0.5
	}

	var newBold float32
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
	fauxRast := self.fauxRenderer.Glyph().GetRasterizer()
	fauxRast.(*mask.FauxRasterizer).SetSkewFactor(self.skewFactor)
}

// Updates the rasterizer's extraWidth.
func (self *Game) refreshBold() {
	fauxRast := self.fauxRenderer.Glyph().GetRasterizer()
	fauxRast.(*mask.FauxRasterizer).SetExtraWidth(self.extraWidth)
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 0, 255 })

	// draw main text
	w, h := screen.Size()
	self.fauxRenderer.Draw(screen, MainText, w/2, h/3)

	// collect helper info
	var info strings.Builder
	info.WriteString(fmt.Sprintf("skew %.2f (%.2f degrees) [right/left]\n", self.skewFactor, -self.skewFactor*45))
	if math.IsNaN(self.italicAngle) {
		info.WriteString("original italic angle unknown\n")
	} else {
		if self.italicAngle == 0 {
			info.WriteString(fmt.Sprintf("original italic angle %.2f degrees\n", self.italicAngle))
		} else {
			info.WriteString(fmt.Sprintf("orig. it. angle %.2f degrees [U unitalicize]\n", self.italicAngle))
		}
	}
	info.WriteString(fmt.Sprintf("bold +%.1fpx [up/down]\n", self.extraWidth))
	if self.quantized {
		info.WriteString("quantization ON [press Q]\n")
	} else {
		info.WriteString("quantization OFF [press Q]\n")
	}
	if self.usingCustomSizer {
		info.WriteString("faux sizer ON [press S]\n")
	} else {
		info.WriteString("faux sizer OFF [press S]\n")
	}
	info.WriteString("[press R to Reset]\n")

	// draw helper info
	self.helpRenderer.Draw(screen, info.String(), w/2, h - h/3)
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

	// create faux rasterizer and sizer
	fauxRast := &mask.FauxRasterizer{}
	customSizer  := &sizer.PaddedAdvanceSizer{}

	// link custom sizer to fauxRast
	fauxRast.SetAuxOnChangeFunc(func(*mask.FauxRasterizer) {
		padding := float64(fauxRast.GetExtraWidth())*0.5 // factor should be between ~[0.5, 1.0]
		customSizer.SetPadding(fract.FromFloat64(padding))
	})

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.Glyph().SetRasterizer(fauxRast)
	renderer.SetSize(36)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)
	renderer.SetColor(color.RGBA{255, 255, 255, 255})
	renderer.Fract().SetHorzQuantization(etxt.QtFull)

	// create helper renderer for other text
	helpRend := etxt.NewRenderer()
	helpRend.Utils().SetCache8MiB() // same underlying cache as "renderer"
	helpRend.SetSize(16)
	helpRend.SetFont(sfntFont)
	helpRend.SetAlign(etxt.Center)
	helpRend.SetColor(color.RGBA{255, 255, 255, 150})

	// get original italic angle information
	postTable := sfntFont.PostTable()
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
		defaultSizer: renderer.GetSizer(),
		italicAngle: italicAngle,
	})
	if err != nil { log.Fatal(err) }
}
