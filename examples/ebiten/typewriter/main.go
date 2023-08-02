package main

import "os"
import "log"
import "fmt"
import "math"
import "image"
import "image/color"
import "math/rand"
import "regexp"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/font"

// This example showcases how to use etxt feeds to create a
// custom and complex text renderer that supports formatting
// markup for changing font color, size, weight, character by
// character uncovering of the text and a few others. Notice
// that this example is only trying to show how to use feeds
// in a non-trivial way, not trying to show how to write the
// best solution to the specific problems it deals with;
// RendererComplex.Draw() can already be used right out of
// the box to solve most of the problems demonstrated in this
// example.
//
// You can run this example with:
//   go run github.com/tinne26/etxt/examples/ebiten/typewriter@latest path/to/font.ttf

const Text = "Hey, hey... are you \\i{there}?\\pause{}\n\nLately, \\#50CB78{color} has been fading out of this world. I don't know where did they send the \\b{original painter}, but the landscape doesn't \\shake{vibrate} quite the same anymore.\\pause{} I dreamed I'd be able to escape from these walls, \\#FFAAAA{resize} the \\#FF3300{virtual room} that tried to contain me for so long and allow my self-expression to continue expanding, but...\n\nThe ever \\bigger{in\\bigger{cr\\bigger{ea\\bigger{si\\bigger{ng}}}}} madness could get to any of us, anytime..\\pause{} We \\#AAAAAA{may not} have prepared properly for it, but it's \\b{\\b{ok}} now.\\pause{}\n\nI didn't give up so easily, though, and travelling through the desert I finally met \\i{\\b{the documentation \\#FF00FF{m}\\#00FFFF{a}\\#FFFF00{s}\\#80FF8F{t}\\#59B487{e}\\#FFC0CB{r}}}, who unveiled some of the secrets I was looking for... we could press \\b{\\b{\\bigger{R}}}, and then... maybe the world itself would vanish from our sights, starting anew in front of a different observer.\n\n\\pause{}An observer that believed to be the same as it always was.\\pause{}\\pause{} Hah.\\pause{}\\pause{} No chance."

// --- typewriter code ---

// - helper types -
const BasicPause  = 4
const PeriodPause = 36
const CommaPause  = 20
const ManualPause = 24

type FormatType int
const (
	FmtSize FormatType = iota
	FmtColor
	FmtBold
	FmtItalic
	FmtPause
	FmtShake
)

type FormatUndo struct {
	formatType FormatType
	data uint64
}

var colorRegexp = regexp.MustCompile(`\A#([0-9A-F]{2})([0-9A-F]{2})([0-9A-F]{2})\z`)

const MaxFormatDepth = 16

// - actual typewriter type -
type Typewriter struct {
	renderer *etxt.Renderer
	content string
	maxIndex int // how far we are into the display of `content`
	pause int // how many pause updates are left before showing the next char
	minPauseIndex int // helper to allow manual pauses
	shaking bool
	backtrack [MaxFormatDepth]FormatUndo
}

func NewTypewriter(font *etxt.Font, size float64, content string) *Typewriter {
	fauxRast := &mask.FauxRasterizer{}
	renderer := etxt.NewRenderer()
	renderer.Glyph().SetRasterizer(fauxRast)
	renderer.SetFont(font)
	renderer.Utils().SetCache8MiB()
	renderer.SetSize(size)
	renderer.SetAlign(etxt.Top | etxt.Left)
	return &Typewriter {
		renderer: renderer,
		content: content,
		pause: PeriodPause,
	}
}

func (self *Typewriter) Reset(content string) {
	self.content = content
	self.maxIndex = 0
	self.shaking = false
	self.pause = BasicPause
}

func (self *Typewriter) Update() {
	self.pause -= 1
	if self.pause <= 0 {
		self.pause = 0
		if self.maxIndex < len(self.content) {
			switch self.content[self.maxIndex] {
			case '.': self.pause = PeriodPause
			case '?': self.pause = PeriodPause
			case ',': self.pause = CommaPause
			default : self.pause = BasicPause
			}
			self.maxIndex += 1
		}
	}
}

func (self *Typewriter) Draw(target *ebiten.Image) {
	bounds := target.Bounds()
	feed := etxt.NewFeed(self.renderer).At(bounds.Min.X, bounds.Min.Y)

	index := 0
	formatDepth := 0
	atLineStart := false

	defer func() {
		for formatDepth > 0 {
			self.undoFormat(self.backtrack[formatDepth - 1])
			formatDepth -= 1
		}
	}()

	for index < self.maxIndex {
		allowStop := true
		fragment, advance := self.nextFragment(index)

		switch fragment[0] {
		case '\\': // apply format
			undo := self.applyFormat(fragment, index)
			self.backtrack[formatDepth] = undo
			formatDepth += 1
			allowStop = false
		case '{': // open braces (only allowed for formats)
			// nothing, the style has already been applied
			allowStop = false
		case '}': // close braces (only allowed for formats)
			undo := self.backtrack[formatDepth - 1]
			self.undoFormat(undo)
			formatDepth -= 1
		case ' ':
			if !atLineStart { feed.Advance(' ') }
		case '\n':
			feed.LineBreak()
			atLineStart = true
		default: // draw text
			// first measure it to see if it fits
			width := self.renderer.Measure(fragment).Width()
			if (feed.Position.X + width).ToIntCeil() > bounds.Max.X {
				feed.LineBreak() // didn't fit, jump to next line
			}

			// abort if we are going beyond the proper text area
			if feed.Position.Y.ToIntCeil() >= bounds.Max.Y { return }

			// draw each character individually
			for i, codePoint := range fragment {
				if index + i >= self.maxIndex { return }
				if self.shaking {
					preY := feed.Position.Y
					vibr := fract.Unit(rand.Intn(96))
					if rand.Intn(2) == 0 { vibr = -vibr }
					feed.Position.Y += vibr
					feed.Draw(target, codePoint)
					feed.Position.Y = preY
				} else {
					feed.Draw(target, codePoint)
				}
			}
			atLineStart = false
		}

		index += advance
		if !allowStop {
			if index >= self.maxIndex && self.maxIndex < len(self.content) {
				self.maxIndex += 1
			}
		}
	}
}

// returns the next fragment and the byte advance
func (self *Typewriter) nextFragment(startIndex int) (string, int) {
	for byteIndex, codePoint := range self.content[startIndex:] {
		switch codePoint {
		case ' ', '\n', '{', '}':
			if byteIndex == 0 {
				return self.content[startIndex : startIndex + 1], 1
			} else {
				return self.content[startIndex : startIndex + byteIndex], byteIndex
			}
		case '\\':
			if byteIndex > 0 {
				return self.content[startIndex : startIndex + byteIndex], byteIndex
			}
		}
	}
	return self.content[startIndex:], len(self.content) - startIndex
}

func (self *Typewriter) applyFormat(format string, index int) FormatUndo {
	if len(format) <= 0 { panic("invalid format with zero length") }
	if format[0] != '\\' { panic("formats must start with backslash, but got '" + format + "'") }
	format = format[1:]
	switch format {
	case "i", "italic", "italics":
		fauxRast := self.renderer.Glyph().GetRasterizer().(*mask.FauxRasterizer)
		factor := fauxRast.GetSkewFactor()
		fauxRast.SetSkewFactor(factor + 0.22)
		return FormatUndo{ FmtItalic, storeFloat64AsUint64(float64(factor)) }
	case "b", "bold":
		fauxRast := self.renderer.Glyph().GetRasterizer().(*mask.FauxRasterizer)
		factor := fauxRast.GetExtraWidth()
		fauxRast.SetExtraWidth(factor + 1.0)
		return FormatUndo{ FmtBold, storeFloat64AsUint64(float64(factor)) }
	case "shake":
		self.shaking = true
		return FormatUndo{ FmtShake, 0 }
	case "pause":
		if self.minPauseIndex <= index {
			self.pause = ManualPause
			self.minPauseIndex = index + 1
		}
		return FormatUndo{ FmtPause, 0 }
	case "bigger":
		size := self.renderer.Fract().GetSize()
		self.renderer.Fract().SetSize(size + 128)
		return FormatUndo{ FmtSize, storeFractAsUint64(size) }
		// note: if we were doing this right, we would have to compute
		//       the whole line in advance, pick the max height and
		//       adjust with that.
	case "smaller":
		size := self.renderer.Fract().GetSize()
		if size > (5*64) {
			self.renderer.Fract().SetSize(size - 128)
		}
		return FormatUndo{ FmtSize, storeFractAsUint64(size) }
	default:
		matches := colorRegexp.FindStringSubmatch(format)
		if matches == nil { panic("unexpected format '" + format + "'") }
		r := parseHexColor(matches[1])
		g := parseHexColor(matches[2])
		b := parseHexColor(matches[3])
		oldColor := self.renderer.GetColor().(color.RGBA)
		self.renderer.SetColor(color.RGBA{r, g, b, 255})
		return FormatUndo{ FmtColor, storeRgbaAsUint64(oldColor) }
	}
}

func (self *Typewriter) undoFormat(undo FormatUndo) {
	switch undo.formatType {
	case FmtSize:
		self.renderer.Fract().SetSize(loadFractFromUint64(undo.data))
	case FmtColor:
		self.renderer.SetColor(loadRgbaFromUint64(undo.data))
	case FmtBold:
		fauxRast := self.renderer.Glyph().GetRasterizer().(*mask.FauxRasterizer)
		fauxRast.SetExtraWidth(float32(loadFloat64FromUint64(undo.data)))
	case FmtShake:
		self.shaking = false
	case FmtPause:
		// nothing to do for this one
	case FmtItalic:
		fauxRast := self.renderer.Glyph().GetRasterizer().(*mask.FauxRasterizer)
		fauxRast.SetSkewFactor(float32(loadFloat64FromUint64(undo.data)))
		// note: if we were doing this right, we would probably want to
		//       consider adding some extra space after italics too in order
		//       to prevent clumping due to italicized portions
	default:
		panic("unexpected format type")
	}
}

// unsafe but fast, already checked with regexp
func parseHexColor(cc string) uint8 {
	return (runeDigit(cc[0]) << 4) + runeDigit(cc[1])
}

// unsafe but fast, already checked with regexp
func runeDigit(r uint8) uint8 {
	if r > '9' { return uint8(r) - 55 }
	return uint8(r) - 48
}

func storeRgbaAsUint64(c color.RGBA) uint64 {
	var u uint64 = uint64(c.R)
	u = (u << 8) | uint64(c.G)
	u = (u << 8) | uint64(c.B)
	return (u << 8) | uint64(c.A)
}
func loadRgbaFromUint64(u uint64) color.RGBA {
	var c color.RGBA
	c.A = uint8(u & 0xFF)
	c.B = uint8((u >> 8) & 0xFF)
	c.G = uint8((u >> 16) & 0xFF)
	c.R = uint8((u >> 24) & 0xFF)
	return c
}
func storeFractAsUint64(f fract.Unit) uint64 { return uint64(uint32(f)) }
func loadFractFromUint64(u uint64) fract.Unit { return fract.Unit(uint32(u)) }
func storeFloat64AsUint64(f float64)  uint64 { return math.Float64bits(f)     }
func loadFloat64FromUint64(u uint64) float64 { return math.Float64frombits(u) }

// --- actual game ---

type Game struct { typewriter *Typewriter }

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.typewriter.renderer.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		self.typewriter.Reset(Text)
	} else {
		self.typewriter.Update()
	}
	return nil
}

func (self *Game) Draw(screen *ebiten.Image) {
	// dark background
	screen.Fill(color.RGBA{ 0, 0, 20, 255 })

	// determine positioning and draw
	w, h := screen.Size()
	scale := ebiten.DeviceScaleFactor()
	offset1 := int(16*scale)
	offset2 := int(32*scale)
	area := image.Rect(offset1, offset1, w - offset2, h - offset2)
	self.typewriter.Draw(screen.SubImage(area).(*ebiten.Image))
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

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/typewriter")
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowResizable(true)
	err = ebiten.RunGame(&Game { NewTypewriter(sfntFont, 18, Text) })
	if err != nil { log.Fatal(err) }
}
