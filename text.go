package etxt

import "math"
import "image/color"
import "unicode/utf8"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// TODO: document somewhere that not popping formatting directives
//       (leaving them active and not clearing them before the end
//       of the Text) is ok, and it won't have any nasty side effects.

// TODO: about text selection and cursor. there are multiple ideas:
//       - handle cursor position automatically and offer callbacks
//         for custom drawing and styling.
//       - offer a low level function for finding the cursor position
//         in a given text (with specific align, location and so on).
//         problem is formatting that later (you kinda have to rebuild
//         the text data, unless we provide a new model for separate
//         index triggered special ops (which could be interesting,
//         actually... may have to consider this more formally)).

// special codes:
// - \x1F : control code, followed by more special values
// - \x00 : after control code, starts glyph run. followed by uint16 glyph run length.
// - \x01 : horz pad, no draw, followed by fract.Unit advance
// - \x02 : custom graphic, +fract.Unit, uint16 for custom data and draw func id
// - \x03 : directive to set text background (e.g. text highlights), +fract.Unit (pad),
//          uint16 data and draw func id
// - \x04 : directive to set text foreground, +fract.Unit (pad), uint16 data and draw func id
// - \x05 : directive to set the text color as RGBA.
// - \x06 : directive to change the active font index.
// [maybe] - \x07 : directive to set the vertical shake function.
// [maybe] - \x08 : arbitrary configuration transform [func(*Renderer, isDraw, isUndo, payload)]
//                     ^ this becomes specially useful now that we have custom draw funcs!
// [maybe] - \x09 : move the start point to a given logical offset... or current position.
// [maybe] - \x10 : set run direction
// [maybe] - \x11 : define glyph pos adjustments
// [maybe] - \x12 : define glyph combination adjustments
// [maybe] - \xNN : fake control code that can be replaced with other codes. useful to reserve space
//                  and replace content later more efficiently. maybe call it "skip?" but this 
//                  would also require direct buffer access to be used efficiently
// [maybe] - \xNK : arbitrary rasterizer or sizer transform? how do I apply those custom changes?
//                  is \x08 enough if I generalize it a bit more?
// TODO: provide some built in functions as control codes. like draw underline or stuff like that?

// A flexible type that can have text content added as utf8, raw
// glyphs or a mix of both, with some styling directives also being
// supported through control codes.
//
// Almost all the methods on this type can be chained:
//   text := etxt.NewText().Add("Hello ").PushColor(cyan).Add("Color").Pop()
//
// Rendering of [Text] requires the use of [RendererComplex] drawing
// functions.
type Text struct {
	buffer []byte
	oscillation float64
	glyphRunLength int
	rollbackStack []uint64 // TODO: hmmm... how did I intend to use this?
}

func NewText() *Text {
	return &Text{ buffer: make([]byte, 0, 32) }
}

type textCancelFormatDirective uint8
const PLF textCancelFormatDirective = 66
const CAF textCancelFormatDirective = 67

// Utility method to dynamically add any kind of content or format you
// want to the Text. It's less type safe and less performant than manually
// calling the static methods, but it's nice to use, especially while
// prototyping and figuring out exactly how you want something to look.
//
// For cancelling format directives, use etxt.[PLF] (pop last formatting
// directive) and etxt.[CAF] (clear all active formatting directives).
func (self *Text) A(args ...any) *Text {
	for _, arg := range args {
		switch typedArg := arg.(type) {
		case string     : _ = self.Add(typedArg)
		case []byte     : _ = self.AddUtf8(typedArg)
		case rune       : _ = self.AddRune(typedArg)
		case color.RGBA : _ = self.PushColor(typedArg)
		case FontIndex  : _ = self.PushFont(typedArg)
		case fract.Unit : _ = self.AddPad(typedArg)
		case []sfnt.GlyphIndex:
			_ = self.AddGlyphs(typedArg)
		case sfnt.GlyphIndex:
			_ = self.AddGlyph(typedArg)
		case textCancelFormatDirective:
			switch typedArg {
			case PLF: _ = self.Pop()
			case CAF: _ = self.PopAll()
			default:
				panic(arg)
			}
		default:
			panic(arg)
		}
	}

	return self
}

func (self *Text) Add(text string) *Text {
	if len(text) == 0 { return self }
	self.ensureGlyphRunClosed()
	self.buffer = append(self.buffer, text...)
	return self
}

func (self *Text) AddRune(codePoint rune) *Text {
	self.ensureGlyphRunClosed()
	self.buffer = utf8.AppendRune(self.buffer, codePoint)
	return self
}

func (self *Text) AddUtf8(bytes []byte) *Text {
	if len(bytes) == 0 { return self }
	self.ensureGlyphRunClosed()
	self.buffer = append(self.buffer, bytes...)
	return self
}

func (self *Text) AddLineBreak() *Text {
	self.ensureGlyphRunClosed()
	self.buffer = append(self.buffer, '\n')
	return self
}

func (self *Text) AddGlyph(index sfnt.GlyphIndex) *Text {
	self.ensureGlyphRunOpen()
	self.appendGlyphIndex(index)
	self.glyphRunLength += 1
	if self.glyphRunLength == 65535 {
		self.ensureGlyphRunClosed()
	}
	return self
}

func (self *Text) AddGlyphs(indices []sfnt.GlyphIndex) *Text {
	self.ensureGlyphRunOpen()

	// notice: we could do some unsafe copying instead,
	// but that would only work as long as we aren't storing
	// the text in game files or caches that may be used
	// across computers with different byte orders. for the
	// moment I prefer the safety, but it can be discussed
	for i := 0; i < len(indices); i++ {
		self.appendGlyphIndex(indices[i])
		self.glyphRunLength += 1
		if self.glyphRunLength == 65535 {
			self.ensureGlyphRunClosed()
			if i == len(indices) - 1 { break }
			self.ensureGlyphRunOpen()
		}
	}
	return self
}

// Adds a logically sized padding spacing. 
func (self *Text) AddPad(logicalPad fract.Unit) *Text {
	f1, f2, f3 := fractToBytes(logicalPad)
	self.buffer = append(self.buffer, []byte{'\x1F', '\x01', f1, f2, f3}...)
	return self
}

func (self *Text) AddGraphic(gfxId uint8, logicalAdvance fract.Unit, payload uint16) *Text {
	f1, f2, f3 := fractToBytes(logicalAdvance)
	p1, p2 := uint8(payload), uint8(payload >> 8)
	self.buffer = append(self.buffer, []byte{'\x1F', '\x02', gfxId, f1, f2, f3, p1, p2}...)
	return self
}

// Formatting directive to set a background draw function.
// To cancel the directive, use [Text.Pop]() or [Text.PopAll]().
func (self *Text) PushBackGFX(gfxId uint8, logicalPad fract.Unit, payload uint16) *Text {
	f1, f2, f3 := fractToBytes(logicalPad)
	p1, p2 := uint8(payload), uint8(payload >> 8)
	self.buffer = append(self.buffer, []byte{'\x1F', '\x03', gfxId, f1, f2, f3, p1, p2}...)
	return self
}

// Formatting directive to set a foreground draw function.
// To cancel the directive, use [Text.Pop]() or [Text.PopAll]().
func (self *Text) PushFrontGFX(gfxId uint8, logicalPad fract.Unit, payload uint16) *Text {
	f1, f2, f3 := fractToBytes(logicalPad)
	p1, p2 := uint8(payload), uint8(payload >> 8)
	self.buffer = append(self.buffer, []byte{'\x1F', '\x04', gfxId, f1, f2, f3, p1, p2}...)
	return self
}

// Formatting directive to alter the text color. To cancel
// the directive, use [Text.Pop]() or [Text.PopAll]().
func (self *Text) PushColor(rgba color.RGBA) *Text {
	self.buffer = append(self.buffer, []byte{'\x1F', '\x05', rgba.R, rgba.G, rgba.B, rgba.A}...)
	return self
}

// Formatting directive to change the active font. To cancel
// the directive, use [Text.Pop]() or [Text.PopAll]().
func (self *Text) PushFont(fontIndex FontIndex) *Text {
	self.buffer = append(self.buffer, []byte{'\x1F', '\x05', uint8(fontIndex)}...)
	return self
}

// func (self *Text) PushMotion(motionType VertMotion, intensityMult float64) *Text {
//
// }

// Returns the current oscillation loop point. The value will
// always be in the range [0, 1). Only relevant when motion
// effecs are used for the text.
func (self *Text) GetOsc() float64 {
	return self.oscillation
}

// Sets the current oscillation loop point. The value should be
// in the range [0, 1). Only relevant when motion effects are
// used for the text.
func (self *Text) SetOsc(point float64) {
	self.oscillation = f64Mod1(point)
}

// Modify the current oscillation loop point by the given shift.
// Only relevant when motion effects are used for the text.
func (self *Text) ShiftOsc(shift float64) {
	self.oscillation = f64Mod1(self.oscillation + shift)
}

// Cancels the nearest active formatting directive.
func (self *Text) Pop() *Text {
	panic("unimplemented")
	return self
}

// Cancels all active formatting directives.
func (self *Text) PopAll() *Text {
	panic("unimplemented")
	return self
}

// Clears the internal contents without deallocating memory.
func (self *Text) Reset() {
	self.buffer = self.buffer[ : 0]
	self.rollbackStack = self.rollbackStack[ : 0]
	self.glyphRunLength = 0
	self.oscillation = 0
}

// ---- internal functions ----

// Closes the currently active glyph run, if any.
func (self *Text) ensureGlyphRunClosed() {
	if self.glyphRunLength <= 0 { return }
	locIndex := len(self.buffer) - (self.glyphRunLength << 1) - 4
	self.buffer[locIndex + 3] = uint8(self.glyphRunLength >> 0)
	self.buffer[locIndex + 4] = uint8(self.glyphRunLength >> 8)
	self.glyphRunLength = 0
}

func (self *Text) ensureGlyphRunOpen() {
	if self.glyphRunLength > 0 { return }
	self.glyphRunLength = 0
	self.buffer = append(self.buffer, []byte{'\x1F', '\x00', '\xFF', '\xFF'}...)
}

func (self *Text) appendGlyphIndex(index sfnt.GlyphIndex) {
	self.buffer = append(self.buffer, uint8(index), uint8(index >> 8))
}

// ---- helpers ----

func fractToBytes(f fract.Unit) (byte, byte, byte) {
	if f.Abs() > 65536*64 - 1 { panic("max fract.Unit absolute value allowed in context is 65535.984375") }
	return uint8(f), uint8(f >> 8), uint8(f >> 16)
}

func fractFromBytes(f1, f2, f3 byte) fract.Unit {
	value := fract.Unit(f1) << 8
	value |= fract.Unit(f2) << 16
	value |= fract.Unit(f3) << 24
	return value >> 8
}
 
func f64Mod1(x float64) float64 {
	_, fract := math.Modf(x)
	if fract < 0 { return fract + 1 }
	return fract
}
