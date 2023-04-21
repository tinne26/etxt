package etxt

import "math"
import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// special codes:
// - \x1F : control code, followed by more special values
// - \x00 : after control code, starts glyph run. followed by uint16 glyph run length.
// - \x01 : horz pad, no draw, followed by fract.Unit advance
// - \x02 : custom graphic, +fract.Unit, uint16 for custom data and draw func id
// - \x03 : directive to set text background, +fract.Unit (pad), uint16 data and draw func id
// - \x04 : directive to set text foreground, +fract.Unit (pad), uint16 data and draw func id
// - \x05 : directive to set the text color as RGBA.
// - \x06 : directive to change the active font index.
// [unimplemented] - \x07 : directive to set the vertical shake function.
// [unimplemented] - \x08 : arbitrary configuration transform [func(*Renderer, isDraw, isUndo, payload)]
// [unimplemented] - \x09 : move the start point to a given logical offset... or current position.
// [unimplemented] - \x10 : set run direction
// [unimplemented] - \x11 : define glyph pos adjustments
// [unimplemented] - \x12 : define glyph combination adjustments
// [unimplemented] - \xNN : fake control code that can be replaced with other codes.
// TODO: provide some built in functions as control codes. like draw underline or stuff like that

// The text may be added as utf8, raw glyphs or a mix of both,
// with some styling directives also supported through control
// codes.
type Text struct {
	buffer []byte
	oscillation float64
	glyphRunLength int
	rollbackStack []uint64
}

func NewText() *Text {
	return &Text{}
}

func (self *Text) Add(text string) *Text {
	if len(text) == 0 { return self }
	self.ensureGlyphRunClosed()
	self.buffer = append(self.buffer, text...)
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

func (self *Text) AddGlyph(index sfnt.GlyphIndex) {
	self.ensureGlyphRunOpen()
	self.appendGlyphIndex(index)
	self.glyphRunLength += 1
	if self.glyphRunLength == 65535 {
		self.ensureGlyphRunClosed()
	}
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

func (self *Text) AddGFX(gfxId uint8, logicalAdvance fract.Unit, payload uint16) *Text {
	f1, f2, f3 := fractToBytes(logicalAdvance)
	p1, p2 := uint8(payload), uint8(payload >> 8)
	self.buffer = append(self.buffer, []byte{'\x1F', '\x02', gfxId, f1, f2, f3, p1, p2}...)
	return self
}

// Formatting directive to set a background draw function.
// To cancel the directive, use [Text.Pop]() or [Text.PopAll]().
func (self *Text) PushBGND(gfxId uint8, logicalPad fract.Unit, payload uint16) *Text {
	f1, f2, f3 := fractToBytes(logicalPad)
	p1, p2 := uint8(payload), uint8(payload >> 8)
	self.buffer = append(self.buffer, []byte{'\x1F', '\x03', gfxId, f1, f2, f3, p1, p2}...)
	return self
}

func (self *Text) PushFGND(gfxId uint8, logicalPad fract.Unit, payload uint16) *Text {
	f1, f2, f3 := fractToBytes(logicalPad)
	p1, p2 := uint8(payload), uint8(payload >> 8)
	self.buffer = append(self.buffer, []byte{'\x1F', '\x04', gfxId, f1, f2, f3, p1, p2}...)
	return self
}

// Formatting directive to alter the text color. To cancel
// the directive, use [Text.Pop]() or [Text.PopAll]().
func (self *Text) PushRGBA(text string, rgba color.RGBA) *Text {
	self.buffer = append(self.buffer, []byte{'\x1F', '\x05', rgba.R, rgba.G, rgba.B, rgba.A}...)
	return self
}

func (self *Text) PushFID(fontId uint8) *Text {
	self.buffer = append(self.buffer, []byte{'\x1F', '\x05', fontId}...)
	return self
}

// func (self *Text) PushMotion(motionType VertMotion, intensityMult float64) *Text {
//
// }

// Only relevant when motion effects are used for the text.
func (self *Text) GetOsc() float64 {
	return self.oscillation
}

func (self *Text) SetOsc(point float64) {
	self.oscillation = f64Mod1(point)
}

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

// Clears the internal buffer without deallocating its memory.
func (self *Text) ClearBuffer() {
	self.buffer = self.buffer[ : 0]
	self.glyphRunLength = 0
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
