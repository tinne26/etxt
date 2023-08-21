package etxt

import "math"
import "strconv"
import "image/color"
import "unicode/utf8"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

type twineCode = uint8
const (
	twineCcBegin twineCode = '\x1F'
	twineCcPop twineCode = '\x00'
	twineCcPopAll twineCode = '\x01'
	twineCcRefreshLineMetrics twineCode = '\x02'
	twineCcSwitchGlyphMode  twineCode = '\x03'
	twineCcSwitchStringMode twineCode = '\x04'

	twineCcPushEffect twineCode = '\x05'
	twineCcPushPreEffect twineCode = '\x06'
	twineCcPushMotion twineCode = '\x07'

	// TODO: text direction, which is another level of trickiness
	// Also consider space earmarking and stop/resume glyph drawing.
	// though stopping is possible with the customFunc, even if rather
	// wasteful.
)

type popSpecialDirective uint8

// Constants for popping special directives on [Weave]().
// Ignore unless using [RendererComplex] and [Twine] values.
const (
	Pop    popSpecialDirective = 66 // pop last pushed directive still active
	PopAll popSpecialDirective = 67 // pop all pushed directives still active
)

// A flexible type that can have text content added as utf8, raw
// glyphs or a mix of both, with some styling directives also being
// supported through control codes and custom functions.
//
// Twines are an alternative to strings relevant for rich text
// formatting, custom effects and text shaping.
//
// Almost all the methods on this type can be chained:
//   var twine etxt.Twine
//   twine.Add("That's ").PushFont(boldIndex).Add("edible").Pop().AddRune('?')
//
// Twines are fairly low level, so writing your own builder types, wrappers
// and functions tailored to your specific use-cases can often be appropriate.
//
// Twine rendering is done through [RendererComplex] drawing functions.
type Twine struct {
	Buffer []byte
	Ticks uint64
	InGlyphMode bool
}

// Creates a [Twine] from the given arguments. For example:
//   rgba  := color.RGBA{ 80, 200, 120, 255 }
//   twine := etxt.Weave("Nice ", rgba, "emerald", etxt.Pop, '!')
//
// You may also pass a twine as the first argument to append to it instead
// of creating a new one. To pop fonts, colors, effects or motions, use the
// etxt.[Pop] and etxt.[PopAll] constants.
func Weave(args ...any) Twine {
	if args == nil { return Twine{ Buffer: make([]byte, 0, 32) } }

	// if first argument is already a twine, we take that 
	// and append to it; otherwise we create a new twine
	ptrTwine, isTwine := args[0].(*Twine)
	if isTwine {
		return *(ptrTwine.Weave(args[1 : ]...))
	} else {
		twine, isTwine := args[0].(Twine)
		if isTwine {
			return *(twine.Weave(args[1 : ]...))
		} else {
			twine = Twine{ Buffer: make([]byte, 0, 32) }
			return *(twine.Weave(args...))
		}
	}
}

// TwineEffectFunc is the function signature for custom functions used
// with [Twine] values and [RendererComplex].
// 
// Effect functions can be triggered in order to change some renderer
// configurations on the fly, render custom graphical layers on top of
// certain text fragments, create primitive strikethrough or underline
// effects, censoring bars, text cursors, highlighting rectangles, padding
// rectangles and many more.
//
// In order to be so flexible, effect functions have to deal with many parameters.
// In most cases, though, you will only be using a few of them at a time. The
// hardest ones to understand are probably the []byte parameter, which is used
// to pass a predefined payload to your function, and the [fract.Unit] return, 
// which tells the renderer how much to advance the pen position in the x axis.
//
// Know also that the payload slice is always a reference to the actual values
// in the [Twine] buffer, so you may even modify them on the fly for your own
// hacky purposes if you are that kind of person.
//
// See also [TwineMotionFunc].
type TwineEffectFunc = func(
	renderer *Renderer, payload []byte,
	flags TwineEffectFlags, target Target,
	origin fract.Point, rect fract.Rect,
) (extraAdvance fract.Unit)

// TwineMotionFunc is a cousin of [TwineEffectFunc] specialized on movement
// animations for text. Some examples are shaking, waving, making text look
// like it's jumping, etc. Unlike effect functions, motion functions are
// called for each glyph, and are skipped while only measuring.
//
// A single twine may use multiple motion functions for different text
// fragments, but only one motion function can be active at a time.
//
// The only tricky parameters are globalOrder and localOrder, which tell
// you how many glyphs have been processed before the current one (globally
// for globalOrder and relative to the current fragment for localOrder).
//
// Notice that a lot of this functionality can also be achieved through
// custom draw functions and [RendererGlyph.SetDrawFunc](), but motion 
// functions are much more practical in many scenarios.
//
// Finally, it's worth pointing out that some motion effects can also be
// created with [TwineEffectFunc], e.g., manipulating the sizer parameters
// in order to control horizontal spacing.
type TwineMotionFunc = func(
	position fract.Point, glyphIndex sfnt.GlyphIndex,
	globalOrder, localOrder int, ticks uint64, payload []byte,
) (xShift, yShift fract.Unit)
// TODO: no closing call? should be ok I guess?
// Also, about stacking. maybe prevent stacking for motion funcs?
// that could make it more manageable

// See [TwineEffectFunc] and [RendererComplex.RegisterEffectFunc]().
// Values above 192 are reserved for internal operation.
type TwineEffectKey uint8
const (
	NextEffectKey TwineEffectKey = 255

	// Basic functions exposed on the Twine API
	EffectPushColor TwineEffectKey = 193 // PushColor()
	EffectPushFont  TwineEffectKey = 194 // PushFont()
	//TwinePad       TwineEffectKey = 195 // expose or not?

	// Advanced functions not directly exposed on the Twine API
	EffectCodeInline TwineEffectKey = 231 // PushEffect(key, nil or []byte{fontIndex, black})
	EffectBackRect TwineEffectKey = 232 // PushPreEffect(key, []byte{r, g, b, a})
	EffectRectOutline TwineEffectKey = 233 // PushEffect(key, nil = []byte{128})
	EffectUnderline TwineEffectKey = 234 // PushEffect(key, nil = []byte{128})
	EffectCrossOut TwineEffectKey = 235 // PushEffect(key, nil = []byte{128})
	EffectSpoiler TwineEffectKey = 236 // PushEffect(key, []byte{black})
	EffectHighlightA TwineEffectKey = 237 // ...
	EffectHighlightB TwineEffectKey = 238 // ...
	EffectHighlightC TwineEffectKey = 239 // ...
	EffectHoverA TwineEffectKey = 240 // ...
	EffectHoverB TwineEffectKey = 241 // ...
	EffectHoverC TwineEffectKey = 242 // ...
	EffectFauxBold TwineEffectKey = 243 // PushEffect(key, nil = []byte{128})
	EffectOblique TwineEffectKey = 244 // PushEffect(key, nil = []byte{192})
	//TwineLineHighlight TwineEffectKey = 245 // yay or nay?
	EffectListItem TwineEffectKey = 246 // PushEffect(key, nil = []byte{128}), uses '-' glyph
	EffectEbi13 TwineEffectKey = 247 // PushEffect(key, nil) + immediate Pop()
	EffectAbbr TwineEffectKey = 248 // PushEffect(key, []byte(tipString))

	// TODO: the most important function missing is
	//       reserving space in the buffer. that can be done
	//       manually too, though, unclear if I should provide
	//       a CC or what. yeah, probably a CC would be the
	//       most efficient approach, though handling that is
	//       still a bit messy on the user side. I don't want
	//       to expose a full API for replacing a reserved area
	//       with active content or whatever.
)

// See [TwineMotionFunc] and [RendererComplex.RegisterMotionFunc]().
// Values above 192 are reserved for internal operation.
type TwineMotionKey uint8
const (
	NextMotionKey TwineMotionKey = 255

	// TODO: implement some nice functions
	MotionVibrate TwineMotionKey = 193 // configure intensity
	MotionShake   TwineMotionKey = 194 // could have many shake types
	MotionWave    TwineMotionKey = 195 // continuous sine wave
	MotionSpooky  TwineMotionKey = 196 // circular movement within a soft sine
	MotionJumpy   TwineMotionKey = 197 // idk if intermittent or not
)

// See [TwineEffectFunc].
type TwineEffectFlags uint8
const (
	TwineTriggerPush      TwineEffectFlags = 0b0000_0001
	TwineTriggerLineBreak TwineEffectFlags = 0b0000_0010
	TwineTriggerLineStart TwineEffectFlags = 0b0000_0100
	TwineTriggerPop       TwineEffectFlags = 0b0000_1000

	TwineFlagPre TwineEffectFlags = 0b1000_0000
	TwineFlagDraw TwineEffectFlags = 0b0100_0000
)

// Returns the TwineTrigger* part of the effect flags.
func (self TwineEffectFlags) GetTrigger() TwineEffectFlags {
	return self & 0b0000_1111
}

// Returns whether the effect is being invoked as a pre effect
// (an effect with lookahead, as opposed to the regular effects).
// See [Twine.PushPreEffect]() vs [Twine.PushEffect]() respectively.
func (self TwineEffectFlags) IsPre() bool {
	return (self & TwineFlagPre) != 0
}

// Returns whether the effect call is happening on a drawing or
// measuring process. While drawing, advances and configuration
// changes that can affect metrics are relevant and must be computed.
// In contrast, when measuring, effects that only change colors or
// other properties that don't affect metrics can be skipped. If you
// are having your logic depend on colors and similar shenanigans...
// try to keep a safe distance from the kids.
//
// Additionally, when measuring, the rendering [Target] will be nil.
func (self TwineEffectFlags) IsDraw() bool {
	return (self & TwineFlagDraw) != 0
}

// The inverse of [TwineEffectFlags.IsDraw]().
func (self TwineEffectFlags) IsMeasure() bool {
	return !self.IsDraw()
}

func assertTwinePayloadBelow256(payload []byte) {
	if len(payload) < 256 { return } // ok
	panic( // not ok
		"Maximum payload size on Twine functions is 255, but got " +
		strconv.Itoa(len(payload)) + " bytes instead",
	)
}

// Chaining methods? Here's your [Weave]() on a [Twine] receiver!
func (self *Twine) Weave(args ...any) *Twine {
	// process each argument
	for _, arg := range args {
		switch typedArg := arg.(type) {
		case string    : _ = self.Add(typedArg)
		case []byte    : _ = self.AddUtf8(typedArg)
		case rune      : _ = self.AddRune(typedArg)
		case FontIndex : _ = self.PushFont(typedArg)
		case TwineEffectKey:
			_ = self.PushEffect(typedArg, nil)
		case TwineMotionKey:
			_ = self.PushMotion(typedArg, nil)
		case color.Color:
			_ = self.PushColor(typedArg)
		case []sfnt.GlyphIndex:
			_ = self.AddGlyphs(typedArg)
		case sfnt.GlyphIndex:
			_ = self.AddGlyph(typedArg)
		case popSpecialDirective:
			switch typedArg {
			case Pop: _ = self.Pop()
			case PopAll: _ = self.PopAll()
			default:
				panic(arg) // invalid popSpecialDirective
			}
		default:
			panic(arg) // invalid argument
		}
	}
	
	return self
}

// Adds the given string to the twine.
func (self *Twine) Add(text string) *Twine {
	self.ensureStringMode()
	if len(text) == 0 { return self }
	self.Buffer = append(self.Buffer, text...)
	return self
}

// Adds the given rune to the twine.
func (self *Twine) AddRune(codePoint rune) *Twine {
	self.ensureStringMode()
	self.Buffer = utf8.AppendRune(self.Buffer, codePoint)
	return self
}

// Adds a line break to the twine. Equivalent to [Twine.AddRune]('\n').
// Mostly useful when working with glyph indices directly, as fonts
// do not contain glyphs for line breaks.
func (self *Twine) AddLineBreak() *Twine {
	return self.AddRune('\n')
}

// Adds the given string bytes to the twine.
func (self *Twine) AddUtf8(bytes []byte) *Twine {
	self.ensureStringMode()
	self.Buffer = append(self.Buffer, bytes...)
	return self
}

// Adds the given glyph to the twine.
func (self *Twine) AddGlyph(index sfnt.GlyphIndex) *Twine {
	self.ensureGlyphMode()
	self.appendGlyphIndex(index)
	return self
}

// Adds the given glyphs to the twine.
func (self *Twine) AddGlyphs(indices []sfnt.GlyphIndex) *Twine {
	self.ensureGlyphMode()

	// notice: we could do some unsafe copying instead,
	// but that would only work as long as we aren't storing
	// the text in game files or caches that may be used
	// across computers with different byte orders. for the
	// moment I prefer the safety, but it could be discussed
	for i := 0; i < len(indices); i++ {
		self.appendGlyphIndex(indices[i])
	}
	return self
}

// Meant to be called once per tick if you have any active
// [TwineMotionFunc] that requires it. If your text changes from 
// frame to frame or you have more advanced needs, you may need to 
// manipulate the [Twine].Ticks field directly or use the motion
// function payload.
func (self *Twine) Tick() {
	self.Ticks += 1
}

// Appends a "pop" directive to the twine. When reached, this directive
// will pop the most recent push directive still active in the twine.
// If no active directives are found, the pop operation will panic.
//
// See also [Twine.PopAll]().
func (self *Twine) Pop() *Twine {
	self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcPop}...)
	return self
}

// Appends a "pop all" directive to the twine. When reached, this
// directive will cancel all push directives still active at the
// current point in the twine.
//
// It's worth noting that leaving special directives active or
// "unpopped" on a twine is perfectly valid; the renderer keeps track
// of that while drawing and will pop any directives left at the end.
func (self *Twine) PopAll() *Twine {
	self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcPopAll}...)
	return self
}

// Clears the internal contents without deallocating memory.
func (self *Twine) Reset() {
	self.Buffer = self.Buffer[ : 0]
	self.Ticks = 0
	self.InGlyphMode = false
}

// Appends a trigger for a [TwineEffectFunc] to the [Twine]. The related
// function, which must be registered with [RendererComplex.RegisterEffectFunc]()
// before the twine is measured or drawn, will remain active until a [Twine.Pop]()
// clears it. The function will be triggered at multiple points:
//  - [TwineTriggerPush] is triggered at the start, with the text not being
//    drawn yet and the [fract.Rect] containing only the pen position.
//    If you need lookahead, see [Twine.PushPreEffect]() instead.
//  - Zero to many sequences of [TwineTriggerLineBreak] and [TwineTriggerLineStart],
//    for each line break. This is necessary because a single rectangle can't
//    properly represent multiple lines.
//  - [TwineTriggerPop] on the next [Twine.Pop]() or at the end of the twine.
func (self *Twine) PushEffect(key TwineEffectKey, payload []byte) *Twine {
	return self.appendKeyWithPayload(twineCcPushEffect, uint8(key), payload)
}

// Appends a trigger for a [TwineEffectFunc] to the [Twine]. This is very
// similar to [Twine.PushEffect](), but with lookahead. Common uses involve
// drawing something behind the text or configuring some properties that
// require knowing the text rect in advance.
// 
// Notice that this procedure makes the [TwineEffectFunc] more expensive
// to use, as multiple passes are necessary.
func (self *Twine) PushPreEffect(key TwineEffectKey, payload []byte) *Twine {
	return self.appendKeyWithPayload(twineCcPushPreEffect, uint8(key), payload)
}

// Similar to [Twine.PushEffect](), but for motion effects. See
// [TwineMotionFunc] for more details.
func (self *Twine) PushMotion(key TwineMotionKey, payload []byte) *Twine {
	return self.appendKeyWithPayload(twineCcPushMotion, uint8(key), payload)
}

// Dealing with text of different sizes and fonts of different families
// within a single paragraph can get very chaotic. As a compromise,
// etxt allows size and font variations within twines, but keeps the
// vertical metrics fixed to the initial (font, size) pair and only
// offers this manual mechanism to refresh the metrics at specific
// points in the text.
//
// Common approaches when size changes are required involve using a
// sizer with an increased line height to account for the maximum
// expected text size.
func (self *Twine) AddLineMetricsRefresh() *Twine {
	self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcRefreshLineMetrics}...)
	return self
}

// ---- common functions provided for utility ----

// Utility function equivalent to [Twine.PushEffect]([EffectPushColor], []byte{r, g, b, a}).
//
// Losses may happen during conversion of textColor to [color.RGBA].
func (self *Twine) PushColor(textColor color.Color) *Twine {
	r, g, b, a := getRGBA8(textColor)
	return self.PushEffect(EffectPushColor, []byte{r, g, b, a})
}

// Utility function equivalent to [Twine.PushEffect]([EffectPushFont], []byte{uint8(index)}).
//
// Before rendering, the font index must have been registered with [RendererComplex.RegisterFont]().
func (self *Twine) PushFont(index FontIndex) *Twine {
	return self.PushEffect(EffectPushFont, []byte{uint8(index)})
}

// ---- internal functions ----

func (self *Twine) ensureGlyphMode() {
	if self.InGlyphMode { return }
	self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcSwitchGlyphMode}...)
	self.InGlyphMode = true
}

func (self *Twine) ensureStringMode() {
	if !self.InGlyphMode { return }
	self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, twineCcSwitchStringMode}...)
	self.InGlyphMode = false
}

func (self *Twine) appendGlyphIndex(index sfnt.GlyphIndex) {
	if index == 0 { // very rare branch
		self.Buffer = append(self.Buffer, 0, 0, 0)
	} else { // common branch
		self.Buffer = append(self.Buffer, uint8(index), uint8(index >> 8))
	}
}

// ---- helpers ----

// used to implement PushEffect, PushPreEffect, PushMotion
func (self *Twine) appendKeyWithPayload(cc, key uint8, payload []byte) *Twine {
	assertTwinePayloadBelow256(payload)
	if self.InGlyphMode {
		self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, cc, key, uint8(len(payload))}...)
	} else {
		self.Buffer = append(self.Buffer, []byte{twineCcBegin, cc, key, uint8(len(payload))}...)
	}
	if len(payload) > 0 {
		self.Buffer = append(self.Buffer, payload...)
	}
	return self
}

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

func getRGBA8(textColor color.Color) (r, g, b, a uint8) {
	rgbaColor, isRGBA := textColor.(color.RGBA)
	if isRGBA {
		return rgbaColor.R, rgbaColor.G, rgbaColor.B, rgbaColor.A
	} else {
		r32, g32, b32, a32 := rgbaColor.RGBA()
		return uint8(r32/65535), uint8(g32/65535), uint8(b32/65535), uint8(a32/65535)
	}
}
