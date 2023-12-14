package main

import "os"
import "log"
import "fmt"
import "math"
import "sort"
import "image"
import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/inpututil"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"

// This example is mostly meant to be an interactive demo on what
// Twines can do. For a more contained code example, check out
// ebiten/examples/twine. The code on this example is long and
// tedious and not very educative.
// 
// You can run the example like this:
//   go run github.com/tinne26/etxt/examples/ebiten/text@latest path/to/font-1.ttf ...
// You can pass additional fonts for this example, as it allows you
// to interactively change the font for different text fragments.

// ---- constants, text samples and colors ----

const BigNumber = 2_000_000_000 // must fit in int32

var textSamples = []string{
	"Visit ebitengine.org for the best cooking tips!", // pen highlight and oblique
	"Oblique text is not that bad,\nbut faux bold can't compete\nagainst actual bold font faces.\n(Use actual bold fonts!)", // oblique, faux bold, small text
	"Golang is not bad. Zig is okay-ish.\nCobol is the past, the present and\npleaaase heeeeelp the future.", // color and cross-out
	"Big, regular, small.\nTake it slow and do not fall.", // sizes
	"Unformatted twine playground.",
}

var defaultFormats = [][]EffectAnnotation{
	[]EffectAnnotation{
		EffectAnnotation{ effectType: EffectPenHighlight, effectParams: []any{highlightColor}, startRune: 6, endRune: 19 },
		EffectAnnotation{ effectType: EffectOblique, startRune: 6, endRune: 19 },
	},
	[]EffectAnnotation{
		EffectAnnotation{ effectType: EffectOblique, effectParams: []any{highlightColor}, startRune: 16, endRune: 27 },
		EffectAnnotation{ effectType: EffectFauxBold, effectParams: []any{highlightColor}, startRune: 33, endRune: 41 },
		EffectAnnotation{ effectType: EffectSetSize, effectParams: []any{SizeOptions[0].Size}, startRune: 87, endRune: 110 },
	},
	[]EffectAnnotation{
		EffectAnnotation{ effectType: EffectSetColor, effectParams: []any{paletteDarkCyan}, startRune: 0, endRune: 5 },
		EffectAnnotation{ effectType: EffectSetColor, effectParams: []any{paletteXanthous}, startRune: 19, endRune: 21 },
		EffectAnnotation{ effectType: EffectSetColor, effectParams: []any{paletteIndianRed}, startRune: 35, endRune: 39 },
		EffectAnnotation{ effectType: EffectCrossOut, startRune: 69, endRune: 85 },
	},
	[]EffectAnnotation{
		EffectAnnotation{ effectType: EffectSetSize, effectParams: []any{SizeOptions[2].Size}, startRune: 0, endRune: 2 },
		EffectAnnotation{ effectType: EffectSetSize, effectParams: []any{SizeOptions[0].Size}, startRune: 14, endRune: 18 },
	},
	[]EffectAnnotation{},
}

var paletteLicorice   = color.RGBA{ 12,   8,  10, 255} // main back color
var paletteIsabelline = color.RGBA{237, 231, 227, 255} // main front color
var paletteDarkCyan   = color.RGBA{ 18, 148, 144, 255}
var paletteIndianRed  = color.RGBA{229,  98,  94, 255}
var paletteMantis     = color.RGBA{123, 201,  80, 255}
var paletteXanthous   = color.RGBA{250, 192,  94, 255}
var paletteCoyote     = color.RGBA{118,  97,  63, 255}
var paletteEmerald    = color.RGBA{ 35, 224, 136, 255}

var backgroundColor = paletteLicorice
var mainTextColor   = paletteIsabelline
var helpTextColor   = rescaleAlpha(paletteIsabelline, 128)
var cursorColor     = rescaleAlpha(paletteEmerald, 240)
var highlightColor  = rescaleAlpha(paletteIndianRed, 156)

// ---- game implementation ----

type Game struct {
	text *etxt.Renderer
	cursorVisible bool
	cursorIndexStart int // line breaks not counted
	cursorIndexEnd int // line breaks not counted
	drawRuneIndex int
	maxDrawRuneIndex int
	textSampleIndex int
	fonts []FontInfo
	effects [][]EffectAnnotation
	effectRecency uint32
	mode InteractionMode
	selectionDir SelDirection
	twine etxt.Twine
	twineAlign etxt.Align
	twineDir etxt.Direction
	auxTwine etxt.Twine
	auxMenuIndex int
	prevMenuIndex int
	vertices [4]ebiten.Vertex
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	var err error
	refreshTwine := false
	
	// common configuration changes
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		self.cursorVisible = !self.cursorVisible
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		switch self.twineAlign {
		case etxt.VertCenter | etxt.Left:
			self.twineAlign = etxt.Center
		case etxt.Center:
			self.twineAlign = etxt.VertCenter | etxt.Right
		case etxt.VertCenter | etxt.Right:
			self.twineAlign = etxt.VertCenter | etxt.Left
		default:
			panic("broken code")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		switch self.twineDir {
		case etxt.LeftToRight: self.twineDir = etxt.RightToLeft
		case etxt.RightToLeft: self.twineDir = etxt.LeftToRight
		default:
			panic("broken code")
		}
	}

	// detect input based on current screen
	switch self.mode {
	case NavigationMode : refreshTwine, err = self.updateNavigation()
	case SelectionMode  : refreshTwine, err = self.updateSelection()
	case InvalidSelectionMode : refreshTwine, err = self.updateInvalidSelection()
	case EffectPickMode : refreshTwine, err = self.updateEffectPick()
	case SizePickMode   : refreshTwine, err = self.updateSizePick()
	case ColorPickMode  : refreshTwine, err = self.updateColorPick()
	case FontPickMode   : refreshTwine, err = self.updateFontPick()
	default:
		panic("broken code")
	}

	if err != nil { return err }

	// apply twine refresh if necessary
	if refreshTwine {
		self.RefreshTwine()
	}

	return nil
}

func (self *Game) updateNavigation() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		self.resetFormatAt(self.textSampleIndex)
		return true, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
			self.textSampleIndex -= 1
			if self.textSampleIndex < 0 {
				self.textSampleIndex = len(textSamples) - 1
			}
		} else {
			self.textSampleIndex += 1
			if self.textSampleIndex >= len(textSamples) {
				self.textSampleIndex = 0
			}
		}
		self.cursorIndexStart = 0
		self.cursorIndexEnd   = 0
		return true, nil
	}
	
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		effectIndex := self.getAreaEffectIndex()
		if effectIndex != -1 {
			effects := self.effects[self.textSampleIndex]
			for i := effectIndex; i < len(effects) - 1; i++ {
				effects[i] = effects[i + 1]
			}
			self.effects[self.textSampleIndex] = effects[ : len(effects) - 1]
			return true, nil
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		self.mode = SelectionMode
		self.selectionDir = SelDirNone
		self.auxMenuIndex = 0
		return false, nil
	}
	
	if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowLeft)) {
		self.cursorIndexStart -= 1
		if self.cursorIndexStart < 0 {
			self.cursorIndexStart = 0
		}
	} else if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowRight)) {
		self.cursorIndexStart += 1
		if self.cursorIndexStart > self.maxDrawRuneIndex {
			self.cursorIndexStart = self.maxDrawRuneIndex
		}
	}
	self.cursorIndexEnd = self.cursorIndexStart
	
	return false, nil // (refreshTwine, error)
}

func (self *Game) updateSelection() (bool, error) {
	// abort selection case
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		self.mode = NavigationMode
		if self.selectionDir == SelDirLeft {
			self.cursorIndexStart = self.cursorIndexEnd
		} else {
			self.cursorIndexEnd = self.cursorIndexStart
		}
		self.selectionDir = SelDirNone
		return false, nil
	}

	// accept selection case
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if self.invalidSelection() {
			self.mode = InvalidSelectionMode
			return false, nil
		} else {
			self.mode = EffectPickMode
			self.refreshEffectPickTwine()
			return false, nil
		}
	}
	
	// adjust selection
	switch self.selectionDir {
	case SelDirNone:
		if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowLeft)) {
			self.cursorIndexStart -= 1
			if self.cursorIndexStart < 0 {
				self.cursorIndexStart = 0
			}
			self.selectionDir = SelDirLeft
		} else if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowRight)) {
			self.cursorIndexEnd += 1
			if self.cursorIndexEnd > self.maxDrawRuneIndex {
				self.cursorIndexEnd = self.maxDrawRuneIndex
			}
			self.selectionDir = SelDirRight
		}
	case SelDirLeft:
		if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowLeft)) {
			self.cursorIndexStart -= 1
			if self.cursorIndexStart < 0 {
				self.cursorIndexStart = 0
			}
		} else if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowRight)) {
			self.cursorIndexStart += 1
			if self.cursorIndexStart == self.cursorIndexEnd {
				self.selectionDir = SelDirNone
			}
		}
	case SelDirRight:
		if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowLeft)) {
			self.cursorIndexEnd -= 1
			if self.cursorIndexEnd == self.cursorIndexStart {
				self.selectionDir = SelDirNone
			}
		} else if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowRight)) {
			self.cursorIndexEnd += 1
			if self.cursorIndexEnd > self.maxDrawRuneIndex {
				self.cursorIndexEnd = self.maxDrawRuneIndex
			}
		}
	default:
		panic("broken code")
	}
	
	return false, nil // (refreshTwine, error)
}

func (self *Game) invalidSelection() bool {
	effects := self.effects[self.textSampleIndex]
	for _, effect := range effects {
		if effect.startRune > self.cursorIndexEnd { continue }
		if self.cursorIndexStart > effect.endRune { continue }
		if effect.startRune >= self.cursorIndexStart && effect.endRune <= self.cursorIndexEnd { continue }
		if self.cursorIndexStart >= effect.startRune && self.cursorIndexEnd <= effect.endRune { continue }
		return true // invalid
	}

	return false
}

func (self *Game) updateInvalidSelection() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		self.mode = SelectionMode
	}
	return false, nil
}

func (self *Game) updateEffectPick() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		self.mode = SelectionMode
		return false, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		effectOpt := EffectOptions[self.auxMenuIndex]
		self.prevMenuIndex = self.auxMenuIndex
		self.auxMenuIndex = 0
		switch effectOpt.Type {
		case EffectSetColor:
			self.mode = ColorPickMode
			self.refreshColorPickTwine()
		case EffectSetSize: // Adjust Size
			self.mode = SizePickMode
			self.refreshSizePickTwine()
		case EffectOblique:
			self.insertEffectAnnotation(EffectAnnotation{
				effectType: EffectOblique,
				startRune: self.cursorIndexStart,
				endRune: self.cursorIndexEnd,
			})
			self.mode = NavigationMode
			return true, nil
		case EffectFauxBold:
			self.insertEffectAnnotation(EffectAnnotation{
				effectType: EffectFauxBold,
				startRune: self.cursorIndexStart,
				endRune: self.cursorIndexEnd,
			})
			self.mode = NavigationMode
			return true, nil
		case EffectSetFont:
			self.mode = FontPickMode
			self.refreshFontPickTwine()
		case EffectPenHighlight:
			// this could have its own color picker, but that indian red is ok
			self.insertEffectAnnotation(EffectAnnotation{
				effectType: EffectPenHighlight,
				effectParams: []any{ highlightColor },
				startRune: self.cursorIndexStart,
				endRune: self.cursorIndexEnd,
			})
			self.mode = NavigationMode
			return true, nil
		case EffectCrossOut:
			self.insertEffectAnnotation(EffectAnnotation{
				effectType: EffectCrossOut,
				startRune: self.cursorIndexStart,
				endRune: self.cursorIndexEnd,
			})
			self.mode = NavigationMode
			return true, nil
		default:
			panic("broken code")
		}
		return false, nil
	}

	if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowUp)) {
		if self.auxMenuIndex > 0 {
			self.auxMenuIndex -= 1
			self.refreshEffectPickTwine()
		}
	} else if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowDown)) {
		if self.auxMenuIndex < len(EffectOptions) - 1 {
			self.auxMenuIndex += 1
			self.refreshEffectPickTwine()
		}
	}

	return false, nil // (refreshTwine, error)
}

type EffectOption struct { Name string ; Type EffectType }
var EffectOptions = []EffectOption{
	EffectOption{"Adjust Color", EffectSetColor},
	EffectOption{"Adjust Size", EffectSetSize},
	EffectOption{"Oblique", EffectOblique},
	EffectOption{"Faux Bold", EffectFauxBold},
	EffectOption{"Change Font", EffectSetFont},
	EffectOption{"Pen Highlight", EffectPenHighlight},
	EffectOption{"Cross-out", EffectCrossOut},
}
func (self *Game) refreshEffectPickTwine() {
	self.auxTwine.Reset()
	for i, opt := range EffectOptions {
		if i == self.auxMenuIndex {
			self.auxTwine.PushColor(cursorColor)
			self.auxTwine.Add("[[ ")
			self.auxTwine.Add(opt.Name)
			self.auxTwine.Add(" ]]")
			self.auxTwine.Pop()
		} else {
			self.auxTwine.Add(opt.Name)
		}
		
		if i != len(EffectOptions) - 1 {
			self.auxTwine.AddLineBreak()
		}
	}
}

func (self *Game) updateColorPick() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		self.mode = EffectPickMode
		self.auxMenuIndex = self.prevMenuIndex
		self.refreshEffectPickTwine()
		return false, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		self.insertEffectAnnotation(EffectAnnotation{
			effectType: EffectSetColor,
			effectParams: []any{ColorOptions[self.auxMenuIndex].Color},
			startRune: self.cursorIndexStart,
			endRune: self.cursorIndexEnd,
		})
		self.mode = NavigationMode
		self.auxMenuIndex = 0 // reset
		return true, nil
	}
	
	if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowUp)) {
		if self.auxMenuIndex > 0 {
			self.auxMenuIndex -= 1
			self.refreshColorPickTwine()
		}
	} else if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowDown)) {
		if self.auxMenuIndex < len(ColorOptions) - 1 {
			self.auxMenuIndex += 1
			self.refreshColorPickTwine()
		}
	}

	return false, nil
}

type ColorOption struct { Name string ; Color color.RGBA }
var ColorOptions = []ColorOption{
	ColorOption{"Isabelline", paletteIsabelline},
	ColorOption{"Mantis", paletteMantis},
	ColorOption{"Xanthous", paletteXanthous},
	ColorOption{"Indian Red", paletteIndianRed},
	ColorOption{"Dark Cyan", paletteDarkCyan},
	ColorOption{"Coyote", paletteCoyote},
}
func (self *Game) refreshColorPickTwine() {
	self.auxTwine.Reset()
	for i, opt := range ColorOptions {
		self.auxTwine.PushColor(opt.Color)
		if i == self.auxMenuIndex {
			self.auxTwine.PushColor(cursorColor)
			self.auxTwine.Add("[[ ")
			self.auxTwine.Pop()
			self.auxTwine.Add(opt.Name)
			self.auxTwine.PushColor(cursorColor)
			self.auxTwine.Add(" ]]")
			self.auxTwine.Pop()
		} else {
			self.auxTwine.Add(opt.Name)
		}
		self.auxTwine.Pop()
		if i != len(ColorOptions) - 1 {
			self.auxTwine.AddLineBreak()
		}
	}
}

func (self *Game) updateSizePick() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		self.mode = EffectPickMode
		self.auxMenuIndex = self.prevMenuIndex
		self.refreshEffectPickTwine()
		return false, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		self.insertEffectAnnotation(EffectAnnotation{
			effectType: EffectSetSize,
			effectParams: []any{SizeOptions[self.auxMenuIndex].Size},
			startRune: self.cursorIndexStart,
			endRune: self.cursorIndexEnd,
		})
		self.mode = NavigationMode
		self.auxMenuIndex = 0 // reset
		return true, nil
	}
	
	if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowUp)) {
		if self.auxMenuIndex > 0 {
			self.auxMenuIndex -= 1
			self.refreshSizePickTwine()
		}
	} else if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowDown)) {
		if self.auxMenuIndex < len(ColorOptions) - 1 {
			self.auxMenuIndex += 1
			self.refreshSizePickTwine()
		}
	}

	return false, nil
}

type SizeOption struct { Name string ; Size uint8 }
var SizeOptions = []SizeOption{
	SizeOption{"Small", 16},
	SizeOption{"Normal", 22},
	SizeOption{"Big", 28},
}
func (self *Game) refreshSizePickTwine() {
	self.auxTwine.Reset()
	for i, opt := range SizeOptions {
		if i == self.auxMenuIndex {
			self.auxTwine.PushColor(cursorColor)
			self.auxTwine.Add("[[ ")
			self.auxTwine.Pop()
			self.auxTwine.Add(opt.Name)
			self.auxTwine.PushColor(cursorColor)
			self.auxTwine.Add(" ]]")
			self.auxTwine.Pop()
		} else {
			self.auxTwine.Add(opt.Name)
		}
		if i != len(ColorOptions) - 1 {
			self.auxTwine.AddLineBreak()
		}
	}
}

func (self *Game) updateFontPick() (bool, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		self.mode = EffectPickMode
		self.auxMenuIndex = self.prevMenuIndex
		self.refreshFontPickTwine()
		return false, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		self.insertEffectAnnotation(EffectAnnotation{
			effectType: EffectSetFont,
			effectParams: []any{self.fonts[self.auxMenuIndex].Index},
			startRune: self.cursorIndexStart,
			endRune: self.cursorIndexEnd,
		})
		self.mode = NavigationMode
		self.auxMenuIndex = 0 // reset
		return true, nil
	}
	
	if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowUp)) {
		if self.auxMenuIndex > 0 {
			self.auxMenuIndex -= 1
			self.refreshFontPickTwine()
		}
	} else if repeat(inpututil.KeyPressDuration(ebiten.KeyArrowDown)) {
		if self.auxMenuIndex < len(self.fonts) - 1 {
			self.auxMenuIndex += 1
			self.refreshFontPickTwine()
		}
	}

	return false, nil
}

func (self *Game) refreshFontPickTwine() {
	self.auxTwine.Reset()
	for i, font := range self.fonts {
		if i == self.auxMenuIndex {
			self.auxTwine.PushColor(cursorColor)
			self.auxTwine.Add("[[ ")
			self.auxTwine.Pop()
			self.auxTwine.Add(font.Name)
			self.auxTwine.PushColor(cursorColor)
			self.auxTwine.Add(" ]]")
			self.auxTwine.Pop()
		} else {
			self.auxTwine.Add(font.Name)
		}
		if i != len(self.fonts) - 1 {
			self.auxTwine.AddLineBreak()
		}
	}
	if len(self.fonts) == 1 {
		self.auxTwine.PushColor(helpTextColor)
		self.auxTwine.Add("\n\n(You need to pass more font paths as arguments\n")
		self.auxTwine.Add("to the program if you want to use additional fonts)")
	}
}

// ---- draw section ----

func (self *Game) Draw(canvas *ebiten.Image) {
	bounds := canvas.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	pad := int(12*ebiten.DeviceScaleFactor())
	
	// fill background
	canvas.Fill(backgroundColor)
	
	// draw general instructions
	var ifstr = func(c bool, a, b string) string {
		if c { return a } else { return b }
	}
	helpText := fmt.Sprintf(
		"[S] Text sample %d/%d\n" + 
		"[V] Cursor %s",
		self.textSampleIndex + 1,
		len(textSamples),
		ifstr(self.cursorVisible, "visible", "hidden"),
	)
	self.text.SetSize(14)
	self.text.SetColor(helpTextColor)
	self.text.SetAlign(etxt.Left | etxt.LastBaseline)
	self.text.Draw(canvas, helpText, pad, h - pad)

	self.text.SetAlign(etxt.HorzCenter | etxt.Baseline)
	if self.mode == NavigationMode {
		self.text.Draw(canvas, "[R] Reset formatting [R]", w/2, h - pad)
	}

	horzAlign := self.twineAlign.Horz().String()
	helpText = fmt.Sprintf(
		"%s [H]\n" + 
		"%s [D]",
		horzAlign[1 : len(horzAlign) - 1],
		self.twineDir.String(),
	)
	self.text.SetAlign(etxt.Right | etxt.LastBaseline)
	self.text.Draw(canvas, helpText, w - pad, h - pad)

	// draw specific interaction mode instructions
	switch self.mode {
	case NavigationMode : self.drawNavigation(canvas, w, h, pad)
	case SelectionMode  : self.drawSelection(canvas, w, h, pad)
	case InvalidSelectionMode : self.drawInvalidSelection(canvas, w, h, pad)
	}

	// draw twine
	self.drawRuneIndex = 0
	self.text.SetAlign(self.twineAlign)
	self.text.SetDirection(self.twineDir)
	self.text.SetSize(22)
	self.text.SetColor(mainTextColor)
	x := self.twineAlign.GetHorzAnchor(w/8, w - w/8)
	self.text.Twine().Draw(canvas, self.twine, x, h/3)
	self.text.SetDirection(etxt.LeftToRight) // restore direction

	// draw overlays for certain modes/screens
	self.text.SetColor(helpTextColor)
	switch self.mode {
	case EffectPickMode : self.drawEffectPick(canvas, w, h)
	case ColorPickMode  : self.drawColorPick(canvas, w, h)
	case SizePickMode   : self.drawSizePick(canvas, w, h)
	case FontPickMode   : self.drawFontPick(canvas, w, h)
	}
}

func (self *Game) drawNavigation(canvas *ebiten.Image, w, h, pad int) {
	self.text.SetSize(16)
	self.text.SetAlign(etxt.Center)
	text := "Use [LEFT] and [RIGHT] to move through the text\n" +
		"Press [ENTER] to start a selection"
	if self.getAreaEffectIndex() != -1 {
		text += "\nPress [DEL] to remove the area's effect"
	}
	self.text.Draw(canvas, text, w/2, h - h/3)
}

func (self *Game) drawSelection(canvas *ebiten.Image, w, h, pad int) {
	self.text.SetSize(16)
	self.text.SetAlign(etxt.Center)
	text := "Use [LEFT] and [RIGHT] to adjust the selection\n" +
		"Press [ESC] or [DEL] to cancel the selection\n"+
		"Press [ENTER] to confirm the selection"
	self.text.Draw(canvas, text, w/2, h - h/3)
}

func (self *Game) drawInvalidSelection(canvas *ebiten.Image, w, h, pad int) {
	self.text.SetSize(16)
	self.text.SetAlign(etxt.Center)
	text := "The current selection is not valid\n" +
		"A selection can't partially overlap other effect areas\n" + 
		"Fully wrapping or being wrapped is allowed\n" +
		"Press [ENTER] or [ESC] or to resume the selection"
	self.text.Draw(canvas, text, w/2, h - h/3)
}

func (self *Game) drawEffectPick(canvas *ebiten.Image, w, h int) {
	self.text.SetSize(16)
	self.text.SetAlign(etxt.Center)
	self.text.Twine().Draw(canvas, self.auxTwine, w/2, h - h/3)
}

func (self *Game) drawColorPick(canvas *ebiten.Image, w, h int) {
	self.text.SetSize(16)
	self.text.SetAlign(etxt.Center)
	self.text.Twine().Draw(canvas, self.auxTwine, w/2, h - h/3)
}

func (self *Game) drawSizePick(canvas *ebiten.Image, w, h int) {
	self.text.SetSize(16)
	self.text.SetAlign(etxt.Center)
	self.text.Twine().Draw(canvas, self.auxTwine, w/2, h - h/3)
}

func (self *Game) drawFontPick(canvas *ebiten.Image, w, h int) {
	self.text.SetSize(16)
	self.text.SetAlign(etxt.Center)
	self.text.Twine().Draw(canvas, self.auxTwine, w/2, h - h/3)
}

func (self *Game) RefreshTwine() {
	// reset twine and push/pop states
	self.twine.Reset()
	effectList := self.effects[self.textSampleIndex]
	for i, _ := range effectList {
		effectList[i].pushed = false
		effectList[i].popped = false
	}

	// iterate text sample to build the twine
	runeCount := 0 
	pushIndex, effect := self.takeNextEffectPush()
	popIndex := self.takeNextEffectPop()
	text := textSamples[self.textSampleIndex]
	for _, codePoint := range text {
		// treat line breaks separately, as we don't
		// want to push/pop effects on them and so on
		if codePoint == '\n' {
			self.twine.AddLineBreak()
			continue
		}

		// push any effects that appear at this point
		for runeCount == pushIndex {
			self.insertEffect(effect)
			pushIndex, effect = self.takeNextEffectPush()
		}

		// add the text rune
		self.twine.AddRune(codePoint)

		// pop any effects that stop at this point
		for runeCount == popIndex {
			self.twine.Pop()
			popIndex = self.takeNextEffectPop()
		}

		// increase rune count
		runeCount += 1
	}

	self.maxDrawRuneIndex = runeCount - 1
}

// Helper for Game.RefreshTwine(), returns the push rune 
// index and the effect type.
func (self *Game) takeNextEffectPush() (int, EffectAnnotation) {
	effectList := self.effects[self.textSampleIndex]
	for i := 0; i < len(effectList); i++ {
		if !effectList[i].pushed {
			effectList[i].pushed = true
			return effectList[i].startRune, effectList[i]
		}
	}
	
	var zeroEffect EffectAnnotation
	return BigNumber, zeroEffect
}

// Helper for Game.RefreshTwine(), returns the pop rune index.
func (self *Game) takeNextEffectPop() int {
	effectList := self.effects[self.textSampleIndex]
	selectedEffectIndex := int(BigNumber)
	earliestPopPosition := int(BigNumber)
	for i, _ := range effectList {
		if !effectList[i].popped {
			if effectList[i].endRune < earliestPopPosition {
				selectedEffectIndex = i
				earliestPopPosition = effectList[i].endRune
			}
		}
	}
	if selectedEffectIndex != BigNumber {
		effectList[selectedEffectIndex].popped = true
	}
	return earliestPopPosition
}

// Helper for Game.RefreshTwine(), modifies Game.twine.
func (self *Game) insertEffect(effect EffectAnnotation) {
	switch effect.effectType {
	case EffectSetColor:
		self.twine.PushColor(effect.effectParams[0].(color.RGBA))
	case EffectSetSize:
		self.twine.PushEffect(etxt.EffectSetSize, etxt.SinglePass, effect.effectParams[0].(uint8))
	case EffectOblique:
		self.twine.PushEffect(etxt.EffectOblique, etxt.SinglePass)
	case EffectFauxBold:
		self.twine.PushEffect(etxt.EffectFauxBold, etxt.SinglePass)
	case EffectSetFont:
		self.twine.PushFont(effect.effectParams[0].(etxt.FontIndex))
	case EffectPenHighlight:
		clr := effect.effectParams[0].(color.RGBA)
		self.twine.PushEffect(etxt.EffectHighlightA, etxt.DoublePass, clr.R, clr.G, clr.B, clr.A)
	case EffectCrossOut:
		self.twine.PushEffect(etxt.EffectCrossOut, etxt.SinglePass)
	default:
		panic(effect.effectType) // unexpected effect type
	}
}

func (self *Game) insertEffectAnnotation(effect EffectAnnotation) {
	// this could be done more efficiently, but this is clearer, not
	// called frequently anyway and the number of effects is contained
	self.effectRecency += 1
	effect.recency = self.effectRecency
	self.effects[self.textSampleIndex] = append(
		self.effects[self.textSampleIndex],
		effect,
	)
	slice := self.effects[self.textSampleIndex]
	sort.SliceStable(slice, func(i, j int) bool {
		if slice[i].startRune < slice[j].startRune { return true }
		if slice[i].startRune > slice[j].startRune { return false }
		// slice[i].startRune == slice[j].startRune
		if slice[i].endRune < slice[j].endRune { return false }
		if slice[i].endRune > slice[j].endRune { return true }
		return slice[i].recency > slice[j].recency
	})
}

func (self *Game) getAreaEffectIndex() int {
	effectList := self.effects[self.textSampleIndex]
	areaEffectIndex := -1
	areaEffectStart := -1
	areaEffectEnd   := BigNumber
	for i, _ := range effectList {
		if effectList[i].OverlapsIndex(self.cursorIndexStart) {
			if effectList[i].startRune > areaEffectStart || effectList[i].endRune < areaEffectEnd {
				areaEffectIndex = i
				areaEffectStart = effectList[i].startRune
				areaEffectEnd   = effectList[i].endRune
			}
		}
	}

	return areaEffectIndex
}

func (self *Game) ResetAllFormats() {
	for i, _ := range defaultFormats {
		self.resetFormatAt(i)
	}
}

func (self *Game) resetFormatAt(n int) {
	self.effects[n] = self.effects[n][ : 0]
	for _, effect := range defaultFormats[n] {
		self.effects[n] = append(self.effects[n], effect)
	}
}

// Custom glyph drawing function that underlines the selected
// glyph if the cursor is visible and in the correct position.
func (self *Game) GlyphDrawFunc(target etxt.Target, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
	var between = func(x, a, b int) bool { return x >= a && x <= b } // inclusive

	mask := self.text.Glyph().LoadMask(glyphIndex, origin)
	self.text.Glyph().DrawMask(target, mask, origin)
	if self.cursorVisible && between(self.drawRuneIndex, self.cursorIndexStart, self.cursorIndexEnd) {
		font   := self.text.GetFont()
		buffer := self.text.GetBuffer()
		size   := self.text.Fract().GetScaledSize()
		advance := self.text.GetSizer().GlyphAdvance(font, buffer, size, glyphIndex)
		ox, oy := origin.ToInts()
		scale := ebiten.DeviceScaleFactor()
		fx := ox + (advance - fract.FromFloat64(1*scale)).ToInt()
		if fx <= ox { fx = ox + 1 }
		thickness := int(4*ebiten.DeviceScaleFactor())
		if thickness < 1 { thickness = 1 }
		rect := image.Rect(ox, oy + 1, fx, oy + thickness)
		fillOver(target.SubImage(rect).(*ebiten.Image), cursorColor)
	}

	self.drawRuneIndex += 1
}

// ---- helper types ----

type SelDirection uint8
const (
	SelDirNone  SelDirection = 0
	SelDirLeft  SelDirection = 1
	SelDirRight SelDirection = 2
)

type EffectAnnotation struct {
	effectType EffectType
	effectParams []any
	startRune int // included
	endRune int // included
	recency uint32
	pushed bool
	popped bool
}
func (self *EffectAnnotation) OverlapsIndex(i int) bool {
	return self.startRune <= i && self.endRune >= i
}

type EffectType uint8
const (
	EffectSetColor EffectType = iota + 1
	EffectSetFont
	EffectFauxBold
	EffectOblique
	EffectSetSize
	EffectBackRect
	EffectPenHighlight
	EffectCrossOut
	EffectUrl
)

type FontInfo struct {
	Name string
	Index etxt.FontIndex
}

type InteractionMode uint8
const (
	NavigationMode InteractionMode = iota
	SelectionMode
	InvalidSelectionMode
	EffectPickMode
	ColorPickMode
	SizePickMode
	FontPickMode
)

// ---- helper functions ----

func rescaleAlpha(rgba color.RGBA, newAlpha uint8) color.RGBA {
	factor := float64(newAlpha)/float64(rgba.A)
	return color.RGBA{
		R: uint8(float64(rgba.R)*factor),
		G: uint8(float64(rgba.G)*factor),
		B: uint8(float64(rgba.B)*factor),
		A: newAlpha,
	}
}

func repeat(tick int) bool {
	return tick == 1 || (tick >= 21 && (tick - 21)%6 == 0)
}

// variables for fillOver function
var vertices [4]ebiten.Vertex
var stdTriOpts ebiten.DrawTrianglesOptions
var mask1x1 *ebiten.Image
func init() {
	mask3x3 := ebiten.NewImage(3, 3)
	mask3x3.Fill(color.White)
	mask1x1 = mask3x3.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	for i := 0; i < len(vertices); i++ {
		vertices[i].SrcX = 1.0
		vertices[i].SrcY = 1.0
	}
}

func fillOver(target *ebiten.Image, fillColor color.Color) {
	bounds := target.Bounds()
	if bounds.Empty() { return }

	r, g, b, a := fillColor.RGBA()
	if a == 0 { return }
	fr, fg, fb, fa := float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535
	for i := 0; i < 4; i++ {
		vertices[i].ColorR = fr
		vertices[i].ColorG = fg
		vertices[i].ColorB = fb
		vertices[i].ColorA = fa
	}

	minX, minY := float32(bounds.Min.X), float32(bounds.Min.Y)
	maxX, maxY := float32(bounds.Max.X), float32(bounds.Max.Y)
	vertices[0].DstX = minX
	vertices[0].DstY = minY
	vertices[1].DstX = maxX
	vertices[1].DstY = minY
	vertices[2].DstX = maxX
	vertices[2].DstY = maxY
	vertices[3].DstX = minX
	vertices[3].DstY = maxY

	target.DrawTriangles(vertices[0 : 4], []uint16{0, 1, 2, 2, 3, 0}, mask1x1, &stdTriOpts)
}

// ---- main / entry point ----

func main() {
	// assert we have enough arguments
	if len(os.Args) < 2 {
		msg := "Usage: expects at least one argument with the path of the " +
		       "fonts to be used, but you can pass up to 9 font paths.\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse main font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Main font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	var rasterizer mask.FauxRasterizer
	renderer.Glyph().SetRasterizer(&rasterizer)
	renderer.Utils().SetCache8MiB()
	renderer.SetFont(sfntFont)

	// parse additional fonts
	var fontInfos []FontInfo
	fontInfos = append(fontInfos, FontInfo{ Name: fontName, Index: renderer.Twine().GetFontIndex() })
	for i := 2; i < len(os.Args); i++ {
		extraFont, extraName, err := font.ParseFromPath(os.Args[i])
		if err != nil { log.Fatal(err) }
		index := renderer.Twine().RegisterFont(etxt.NextFontIndex, extraFont)
		fontInfos = append(fontInfos, FontInfo{ Name: extraName, Index: index })
		fmt.Printf("Loaded additional font: %s\n", extraName)
	}

	// create game interface and initialize
	game := &Game{
		text: renderer,
		fonts: fontInfos,
		effects: make([][]EffectAnnotation, len(textSamples)),
		cursorVisible: true,
		twineDir: etxt.LeftToRight,
		twineAlign: etxt.Center,
	}
	game.ResetAllFormats()
	game.RefreshTwine()
	renderer.Glyph().SetDrawFunc(game.GlyphDrawFunc)

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/twine_demo")
	ebiten.SetWindowSize(640, 480)	
	ebiten.SetScreenClearedEveryFrame(false)
	err = ebiten.RunGame(game)
	if err != nil { log.Fatal(err) }
}
