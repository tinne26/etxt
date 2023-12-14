package etxt

import "math"
import "strconv"
import "image/color"
import "unicode/utf8"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// TODO:
// List of underspecified cases:
// - handling of negative advances going beyond start of line
// - after pads, positions might be unquantized. hmm. it's not technically wrong,
//   but it feels weird that quantization is enabled and yet we can call the effect
//   functions with non-quantized positions. make sure it doesn't lead to bugs

type twineCode = uint8
const (
	twineCcBegin twineCode = '\x1F' // can't be 0
	twineCcPop twineCode = '\x00'
	twineCcPopAll twineCode = '\x01'
	twineCcStop twineCode = '\x02'
	twineCcRefreshLineMetrics twineCode = '\x03'
	twineCcSwitchGlyphMode  twineCode = '\x04'
	twineCcSwitchStringMode twineCode = '\x05'

	twineCcPushSinglePassEffect twineCode = '\x06'
	twineCcPushDoublePassEffect twineCode = '\x07'
	twineCcPushEffectWithSpacing twineCode = '\x08'
	twineCcPushMotion twineCode = '\x09'

	twineCcPushLineRestartMarker twineCode = '\x0A'
	twineCcPopLineRestartMarker twineCode = '\x0B'

	// notes:
	// - consider space earmarking and stop/resume glyph drawing.
	//   though stopping is possible with the customFunc, even if 
	//   rather wasteful.
	// - won't add changeable text dir mid twine, too annoying
)

type popSpecialDirective uint8

// Constants for popping special directives when working with
// [Weave]() and [Twine.Weave]().
const (
	Pop    popSpecialDirective = 66 // pop last effect function still active
	PopAll popSpecialDirective = 67 // pop all effect functions still active
	Stop   popSpecialDirective = 68 // pop last motion function still active
)

// A flexible type that can have text content added as utf8, raw
// glyphs or a mix of both, with some styling directives also being
// supported through control codes and custom functions.
//
// Twines are an alternative to strings relevant for text formatting,
// custom effects and direct glyph encoding.
//
// Almost all the methods on this type can be chained:
//   var twine etxt.Twine
//   twine.Add("Is it ").PushFont(boldIndex).Add("edible").Pop().AddRune('?')
//
// Twines are somewhat low level, so writing your own builder types, wrappers
// and tailored functions can often be appropriate.
//
// Twine rendering is done through [RendererTwine.Draw]().
type Twine struct {
	Buffer []byte
	Ticks uint64
	InGlyphMode bool // we start in utf8 mode, not glyph mode
}

// Creates a [Twine] from the given arguments. For example:
//   rgba  := color.RGBA{ 80, 200, 120, 255 }
//   twine := etxt.Weave("Nice ", rgba, "emerald", etxt.Pop, '!')
//
// You can also pass a twine as the first argument to append to it instead
// of creating a new one. To pop fonts, colors, effects or motions, use the
// etxt.[Pop], etxt.[PopAll] and etxt.[Stop] constants.
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

// TwineEffectFunc is the signature for custom effect functions
// that can be used within twines.
// 
// Effect functions can be triggered in order to change some renderer
// configurations on the fly, render custom graphical layers on top of
// certain text fragments, create primitive strikethrough or underline
// effects, censoring bars, text cursors, highlighting rectangles and
// many others.
//
// In order to be so flexible, effect functions have to deal with a fair
// amount of parameters and different call points. We structure these
// through [TwineEffectArgs].
//
// See also [TwineMotionFunc].
type TwineEffectFunc = func(renderer *Renderer, target Target, args TwineEffectArgs)

// Related to [TwineEffectFunc] and [RendererTwine.RegisterEffectFunc]().
// Values above 192 are reserved for internal operation.
type TwineEffectKey uint8
const (
	NextEffectKey TwineEffectKey = 255

	// Basic functions exposed on the Twine API
	EffectPushColor TwineEffectKey = 193 // PushColor()
	EffectPushFont TwineEffectKey = 194 // PushFont()
	EffectShiftSize TwineEffectKey = 195 // ShiftSize()

	// Advanced functions not directly exposed on the Twine API
	EffectSetSize TwineEffectKey = 230 // SinglePass + uint8 size
	EffectCodeInline TwineEffectKey = 231 // [UNIMPLEMENTED] SinglePass + (fontIndex, color) or nil (= []byte{fontIndex, black})
	EffectBackRect TwineEffectKey = 232 // [UNIMPLEMENTED] DoublePass + []byte{r, g, b, a} (alpha is optional)
	EffectRectOutline TwineEffectKey = 233 // [UNIMPLEMENTED] SinglePass + relThickness or nil (= []byte{128})
	EffectRawUnderline TwineEffectKey = 234 // [UNIMPLEMENTED] SinglePass + relThickness or nil (= []byte{128})
	EffectCrossOut TwineEffectKey = 235 // SinglePass + relThickness or nil (= []byte{128})
	EffectSpoiler TwineEffectKey = 236 // [UNIMPLEMENTED] SinglePass + color or nil (= []byte{black})
	EffectHighlightA TwineEffectKey = 237 // DoublePass + []byte{r, g, b, a} (alpha is optional)
	EffectHighlightB TwineEffectKey = 238 // [UNIMPLEMENTED] DoublePass + []byte{r, g, b, a} (alpha is optional)
	EffectHighlightC TwineEffectKey = 239 // [UNIMPLEMENTED] DoublePass + []byte{r, g, b, a} (alpha is optional)
	EffectHoverA TwineEffectKey = 240 // [UNIMPLEMENTED] DoublePass + []byte{r, g, b, a} (alpha is optional)
	EffectHoverB TwineEffectKey = 241 // [UNIMPLEMENTED] DoublePass + []byte{r, g, b, a} (alpha is optional)
	EffectHoverC TwineEffectKey = 242 // [UNIMPLEMENTED] DoublePass + []byte{r, g, b, a} (alpha is optional)
	EffectFauxBold TwineEffectKey = 243 // SinglePass + relThickness or nil (= []byte{128})
	EffectOblique TwineEffectKey = 244 // SinglePass + relSkew or nil (= []byte{192})
	EffectListItem TwineEffectKey = 246 // [UNIMPLEMENTED] SinglePass [uses '-' glyph]
	EffectEbi13 TwineEffectKey = 247 // [UNIMPLEMENTED] SinglePass, expects immediate Pop()
	EffectAbbr TwineEffectKey = 248 // [UNIMPLEMENTED] PushEffect(key, []byte(tipString))

	// TODO: the most important function missing is probably
	//       reserving space in the buffer. that can be done
	//       manually too, though, unclear if I should provide
	//       a CC or what. yeah, probably a CC would be the
	//       most efficient approach, though handling that is
	//       still a bit messy on the user side. I don't want
	//       to expose a full API for replacing a reserved area
	//       with active content or whatever.
)

// Related to [TwineEffectFunc].
// 
// Effects can be used in two different modes: 
//  - Single pass mode: the effect function will be invoked at the starting
//    and ending points with whatever information is available at the moment.
//  - Double pass mode: whenever we try to draw with the effect, we will
//    perform an initial pass measuring the text content within the span
//    of the effect so its width is known at rendering time. This is more
//    expensive, but some effects require it in order to draw additional
//    graphics behind the text.
type TwineEffectMode bool
const (
	SinglePass TwineEffectMode = true
	DoublePass TwineEffectMode = false
)
func (self TwineEffectMode) string() string {
	switch self {
	case SinglePass: return "SinglePass"
	case DoublePass: return "DoublePass"
	default:
		return "InvalidTwineEffectMode"
	}
}

func (self TwineEffectMode) controlCode() byte {
	switch self {
	case SinglePass: return twineCcPushSinglePassEffect
	case DoublePass: return twineCcPushDoublePassEffect
	default:
		panic("invalid effect mode")
	}
}

// Twine effect triggers are returned by [TwineEffectArgs.GetTrigger]()
// and indicate the [TwineEffectFunc] invocation reason.
//
// The lifetime of an effect goes like this, for each line:
//  - The effect is invoked with [TwineTriggerPush].
//  - If the effect remains active beyond the end of the line, it will
//    be invoked with [TwineTriggerLineBreak] and then again with
//    [TwineTriggerLineStart] soon after.
//  - When the effect is popped or the text ends, the effect is invoked
//    one last time with [TwineTriggerPop].
// Notice that while drawing, this sequence will always happen at least
// once in draw mode, but could also happen one additional time, before
// drawing, in measuring mode. During measuring processes, instead, only
// measuring calls will be issued.
type TwineEffectTrigger uint8
const (
	TwineTriggerPush      TwineEffectTrigger = 0b0000_0001
	TwineTriggerLineStart TwineEffectTrigger = 0b0000_0010
	TwineTriggerLineBreak TwineEffectTrigger = 0b0000_0100
	TwineTriggerPop       TwineEffectTrigger = 0b0000_1000
)

const (
	twineFlagMeasuring   uint8 = 0b0100_0000
	twineFlagDoublePass  uint8 = 0b1000_0000
	twineFlagRightToLeft uint8 = 0b0010_0000
)

// TwineEffectArgs are used to communicate the conditions under which
// a [TwineEffectFunc] is invoked. Twine effect functions can be called 
// while measuring, drawing, on line breaks, etc. This is admittedly
// not easy; you will have to go through the documentation slowly
// putting all the pieces together.
//
// For the moment, here's a general description of the public fields:
//  - Payload: can be used to pass some predefined values to an effect.
//    The slice is always a reference to the actual values in the [Twine]
//    buffer, so you may even modify them on the fly for your own hacky
//    purposes (if you are that kind of person).
//  - Origin: the pen position where the effect started. If the effect
//    spans multiple lines, the origin will be reset to the start of
//    each new one. The PrePad, if any, extends behind the origin X.
//  - Ascent, Descent: low-level metrics. They indicate the ascent/descent of
//    the current line as absolute values. See [Twine.AddLineMetricsRefresh]()
//    for more details on how line metrics and size changes work with twines.
//  - KnownWidth: the maximum known width of the content within the scope of
//    the effect. On effect pops, line breaks and double-pass effect draw 
//    passes, this will match the actual width of the text. If a minimum content
//    width is set, that will be reflected too. In any other case, the max known
//    width remains unknown. In case of multiple lines, the value only reflects
//    the width within the current line.
//  - PrePad: optional effect padding specified through [Twine.PushEffectWithSpacing]().
//    Notice that pre padding values can change based on whether the effect is
//    beginning or wrapping into a new line.
//  - KnownPostPad: optional effect padding similar to PrePad, with the difference
//    that the value is only known in the same circumstances as KnownWidth.
type TwineEffectArgs struct {
	Payload []byte
	Origin fract.Point // unreliable while measuring
	LineAscent fract.Unit // remains constant throughout the whole line, ...
	LineDescent fract.Unit // ... might not match the *current* font size
	KnownWidth fract.Unit // as an absolute value. see documentation for details
	PrePad fract.Unit // as an absolute value. see Twine.PushEffectWithSpacing()
	KnownPostPad fract.Unit // as an absolute value. see Twine.PushEffectWithSpacing()
	flags uint8

	// NOTE: we could use "KnownAdvance" instead of width, but even if we
	// have to expand and add some "CcBreakpoint" (manual trigger) that receives
	// the function key and maybe a breakpoint value, we would still need an extra
	// "Advance/X"
}

// Returns the reason for the current [TwineEffectFunc] invocation.
// See [TwineEffectTrigger] for further details.
func (self *TwineEffectArgs) GetTrigger() TwineEffectTrigger {
	return TwineEffectTrigger(self.flags & 0b0001_1111)
}

// Returns the current effect mode: [SinglePass] or [DoublePass].
// The effect mode is determined by the value passed to the original
// [Twine.PushEffect]().
//
// Double pass effects are relevant when you need to draw behind
// the text or know the text fragment size in advance, among others.
func (self *TwineEffectArgs) Mode() TwineEffectMode {
	return TwineEffectMode((self.flags & twineFlagDoublePass) == 0)
}

// Returns whether the effect is part of a [LeftToRight] (true)
// or [RightToLeft] (false) operation.
//
// Many effects need to take into account whether the Origin field
// starts on the left or the right side of the word before drawing.
func (self *TwineEffectArgs) IsLeftToRight() bool {
	return (self.flags & twineFlagRightToLeft) == 0
}

// Utility method to return the [fract.Rect] of the text within the scope
// of the effect. The rect only considers the current line, and the
// width is derived from KnownWidth. If KnownWidth is not known at the
// time of using this method, the return value is meaningless. 
//
// Related: [TwineEffectArgs.AreMetricsKnown]().
func (self *TwineEffectArgs) Rect() fract.Rect {
	if self.IsLeftToRight() {
		return fract.UnitsToRect(
			self.Origin.X                    , // x min
			self.Origin.Y - self.LineAscent  , // y min
			self.Origin.X + self.KnownWidth  , // x max
			self.Origin.Y + self.LineDescent , // y max
		)
	} else { // RightToLeft
		return fract.UnitsToRect(
			self.Origin.X - self.KnownWidth  , // x min
			self.Origin.Y - self.LineAscent  , // y min
			self.Origin.X                    , // x max
			self.Origin.Y + self.LineDescent , // y max
		)
	}
}

// Utility method similar to [TwineEffectArgs.Rect](), but also
// taking pre and post paddings into consideration.
func (self *TwineEffectArgs) RectWithPads() fract.Rect {
	if self.IsLeftToRight() {
		return fract.UnitsToRect(
			self.Origin.X - self.PrePad                         , // x min
			self.Origin.Y - self.LineAscent                     , // y min
			self.Origin.X + self.KnownWidth + self.KnownPostPad , // x max
			self.Origin.Y + self.LineDescent                    , // y max
		)
	} else {
		return fract.UnitsToRect(
			self.Origin.X - self.KnownWidth - self.KnownPostPad , // x min
			self.Origin.Y - self.LineAscent                     , // y min
			self.Origin.X + self.PrePad                         , // x max
			self.Origin.Y + self.LineDescent                    , // y max
		)
	}
}

// Returns whether we are in a context in which the KnownWidth and
// KnownPostPad are fully known.
//
// Notice that you can generally know this without calling the method
// explicitly. The situations in which the metrics are available are
// already described in [TwineEffectArgs]. That being said, sometimes
// it's simpler and/or safer to check explicitly.
func (self *TwineEffectArgs) AreMetricsKnown() bool {
	return (self.GetTrigger() > TwineTriggerLineStart) ||
	       (self.Drawing() && self.Mode() == DoublePass)
}

// Returns whether the effect function is being called while drawing
// or measuring. While not drawing, effects that only change colors
// or other properties that don't affect metrics can generally return
// early.
//
// See also [TwineEffectArgs.Measuring]().
func (self *TwineEffectArgs) Drawing() bool {
	return (self.flags & twineFlagMeasuring) == 0
}

// The inverse of [TwineEffectArgs.Drawing]().
func (self *TwineEffectArgs) Measuring() bool {
	return (self.flags & twineFlagMeasuring) != 0
}

// Panics if the payload length differs from the given value.
func (self *TwineEffectArgs) AssertPayloadLen(numBytes int) {
	if len(self.Payload) == numBytes { return }
	
	assert := strconv.Itoa(numBytes)
	actual := strconv.Itoa(len(self.Payload))
	panic("TwineEffectFunc expects a payload of " + assert + " bytes, but got " + actual + " instead.")
}

// Panics if the effect mode doesn't match the given one.
func (self *TwineEffectArgs) AssertMode(effectMode TwineEffectMode) {
	if self.Mode() == effectMode { return }
	
	// panic tip: did you set the proper mode on Twine.PushEffect(etxt.DoublePass, ...)?
	switch effectMode {
	case SinglePass: panic("expected TwineEffectArgs.Type() == etxt.SinglePass")
	case DoublePass: panic("expected TwineEffectArgs.Type() == etxt.DoublePass")
	default:
		panic("invalid effect mode")
	}
}

// [NOTICE: MOTION FUNCTIONS ARE NOT IMPLEMENTED ON THE DRAWING ALGORITHMS YET]
// 
// TwineMotionFunc is a cousin of [TwineEffectFunc] specialized on movement
// animations for text. Some examples are shaking, waving, making text look
// like it's jumping, etc. Unlike effect functions, motion functions are
// called for each glyph and can't affect measuring operations.
//
// A single twine may use multiple motion functions for different text
// fragments, but only one motion function can be active at a time. On
// the upside, motion funcs can overlap or intersect effect functions.
//
// Notice that most of this functionality can also be reproduced with
// custom draw functions and [RendererGlyph.SetDrawFunc](), but motion 
// functions are much more practical in many scenarios.
type TwineMotionFunc = func(
	position fract.Point, glyphIndex sfnt.GlyphIndex,
	order TwineMotionOrder, ticks uint64, payload []byte,
) (xShift, yShift fract.Unit)

// Related to [TwineMotionFunc]. The twine motion order struct tells
// us how many glyphs have been processed before the current one.
type TwineMotionOrder struct {
	Text int // glyph order within the whole text
	Line int // glyph order within the current line (automatic line wrapping resets this too)
	Fragment int // glyph order within the current fragment (can span multiple lines)
}

// See [TwineMotionFunc] and [RendererTwine.RegisterMotionFunc]().
// Values above 192 are reserved for internal operation.
type TwineMotionKey uint8
const (
	NextMotionKey TwineMotionKey = 255

	// A few basic and nice motion functions. [UNIMPLEMENTED]
	MotionVibrate TwineMotionKey = 193 // configure intensity
	MotionShake   TwineMotionKey = 194 // could have many shake types
	MotionWave    TwineMotionKey = 195 // continuous sine wave
	MotionSpooky  TwineMotionKey = 196 // circular movement within a soft sine
	MotionJumpy   TwineMotionKey = 197 // idk if intermittent or not
	MotionGlitchy TwineMotionKey = 198 // random jumpiness in random places
	// TODO: motions that apply two separate intensity functions for
	// odd and even glyphs are interesting, look into that.
)

// [Weave]() on a [Twine] receiver. Useful when chaining methods.
func (self *Twine) Weave(args ...any) *Twine {
	// process each argument
	for _, arg := range args {
		switch typedArg := arg.(type) {
		case string    : _ = self.Add(typedArg)
		case []byte    : _ = self.AddUtf8(typedArg)
		case rune      : _ = self.AddRune(typedArg)
		case FontIndex : _ = self.PushFont(typedArg)
		case TwineEffectKey:
			_ = self.PushEffect(typedArg, SinglePass)
		case TwineMotionKey:
			_ = self.Move(typedArg, nil)
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
			case Stop: _ = self.Stop()
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
// This method is particularly helpful when working directly with glyph
// indices, as fonts do not contain glyphs for control codes.
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
// manipulate the Twine.Ticks field directly or use the motion
// function payload.
func (self *Twine) Tick() {
	self.Ticks += 1
}

// Appends a "pop" directive to the twine. When reached, this directive
// will pop the most recent effect push directive still active in the twine.
// If no active directives are found, the pop operation will panic.
//
// See also [Twine.PopAll]().
func (self *Twine) Pop() *Twine {
	if self.InGlyphMode { // ctrl+f [#1] for more details
		self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, twineCcPop}...)
	} else {
		self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcPop}...)
	}
	return self
}

// Appends a "pop all" directive to the twine. When reached, this
// directive will cancel all effect push directives still active at 
// the current point in the twine. See also [Twine.Pop]().
//
// It's worth noting that leaving special directives active or
// "unpopped" on a twine is perfectly valid; the renderer automatically
// issues a "pop all" for any directives left at the end. 
func (self *Twine) PopAll() *Twine {
	if self.InGlyphMode { // ctrl+f [#1] for more details
		self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, twineCcPopAll}...)
	} else {
		self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcPopAll}...)
	}
	return self
}

// The equivalent to [Twine.Pop]() for motion functions. Motion functions
// don't have a "pop all" equivalent because only one can be active at a
// time.
func (self *Twine) Stop() *Twine {
	if self.InGlyphMode { // ctrl+f [#1] for more details
		self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, twineCcStop}...)
	} else {
		self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcStop}...)
	}
	return self
}

// Clears the internal contents of the twine without deallocating memory.
func (self *Twine) Reset() {
	self.Buffer = self.Buffer[ : 0]
	self.Ticks = 0
	self.InGlyphMode = false
}

// Appends a trigger for a [TwineEffectFunc] to the [Twine]. The related
// function, which must be registered with [RendererTwine.RegisterEffectFunc]()
// before the twine is measured or drawn, will remain active until a [Twine.Pop]()
// clears it.
func (self *Twine) PushEffect(key TwineEffectKey, effectMode TwineEffectMode, payload ...byte) *Twine {
	return self.appendKeyWithPayload(effectMode.controlCode(), uint8(key), payload)
}

// Similar to [Twine.PushEffect](), but for motion functions.
// Unlike effect functions, motion functions can't be nested.
// See [TwineMotionFunc] for more details.
func (self *Twine) Move(key TwineMotionKey, payload []byte) *Twine {
	return self.appendKeyWithPayload(twineCcPushMotion, uint8(key), payload)
}

// Dealing with text of different sizes and fonts of different families
// within a single paragraph can get very chaotic. As a compromise,
// etxt allows size and font variations within twines, but keeps the
// vertical metrics fixed to the initial (font, size, sizer) combination
// and only offers this manual mechanism to refresh the metrics at
// specific points in the text.
//
// Common approaches when size changes are required involve using a
// [sizer.Sizer] with an increased line height to account for the 
// maximum expected text size.
// 
// Notice that refreshed line metrics won't become effective until the
// next line break. Refreshes directly after a new line might not work
// as you expect. This doesn't seem very user-friendly, but all the
// user-friendly options have some dark side to them. For the moment
// I prefer to stick with the awkward but explicit behavior.
//
// As you can see, this is currently a very low-level precision tool.
func (self *Twine) AddLineMetricsRefresh() *Twine {
	if self.InGlyphMode { // ctrl+f [#1] for more details
		self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, twineCcRefreshLineMetrics}...)
	} else {
		self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcRefreshLineMetrics}...)
	}
	return self
}

// ---- advance and padding tricks ----

// Same as [Twine.PushEffect]() but including additional spacing information.
// See [TwineEffectSpacing] for more details on paddings, minimum width and
// so on.
func (self *Twine) PushEffectWithSpacing(key TwineEffectKey, mode TwineEffectMode, spacing TwineEffectSpacing, payload ...byte) *Twine {
	cc := mode.controlCode()
	self.assertValidPayloadLen(payload...)
	if self.InGlyphMode {
		self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, twineCcPushEffectWithSpacing}...)
	} else {
		self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcPushEffectWithSpacing}...)
	}
	self.Buffer = spacing.appendData(self.Buffer)
	self.Buffer = append(self.Buffer, []byte{cc, uint8(key), uint8(len(payload))}...)
	if len(payload) > 0 { self.Buffer = append(self.Buffer, payload...) }
	return self
}

// Registers the current horizontal position in the text and sets it as
// the new line restart position. This is useful to create itemized
// lists or any other kind of text block that requires indentation for
// multiple lines.
//
// Line restart markers can be removed with [Twine.PopLineRestartMarker]().
func (self *Twine) PushLineRestartMarker() *Twine {
	if self.InGlyphMode {
		self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, twineCcPushLineRestartMarker}...)
	} else {
		self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcPushLineRestartMarker}...)
	}
	return self
}

// Removes the most recent line restart marker that's still active.
// If no line restart markers remain, the pop will panic.
// 
// See also [Twine.PushLineRestartMarker]().
// 
// Notice that [Twine.Pop](), despite the name similarity, doesn't
// affect line restart markers.
func (self *Twine) PopLineRestartMarker() *Twine {
	if self.InGlyphMode {
		self.Buffer = append(self.Buffer, []byte{0, 0, twineCcBegin, twineCcPopLineRestartMarker}...)
	} else {
		self.Buffer = append(self.Buffer, []byte{twineCcBegin, twineCcPopLineRestartMarker}...)
	}
	return self
}

// ---- common functions provided for utility ----

// Utility function equivalent to [Twine.PushEffect]([EffectPushColor],
// [SinglePass], []byte{r, g, b, a}).
//
// Losses may happen during conversion of textColor to [color.RGBA].
func (self *Twine) PushColor(textColor color.Color) *Twine {
	r, g, b, a := getRGBA8(textColor)
	return self.PushEffect(EffectPushColor, SinglePass, []byte{r, g, b, a}...)
}

// Utility function equivalent to [Twine.PushEffect]([EffectPushFont],
// [SinglePass], []byte{uint8(index)}).
//
// Before rendering, the font index must have been registered with
// [RendererTwine.RegisterFont]().
func (self *Twine) PushFont(index FontIndex) *Twine {
	return self.PushEffect(EffectPushFont, SinglePass, uint8(index))
}

// Utility function equivalent to [Twine.PushEffect]([EffectShiftSize],
// [SinglePass], []byte{uint8(logicalSizeChange)}).
//
// Size changes operate under special rules detailed on [Twine.AddLineMetricsRefresh]().
func (self *Twine) ShiftSize(logicalSizeChange int8) *Twine {
	return self.PushEffect(EffectShiftSize, SinglePass, uint8(logicalSizeChange))
}

// ---- internal functions ----
// Internal note [#1]:
// Glyph index 0, which corresponds to "notdef", is rarely used, and we take
// advantage of this when encoding control codes in glyph mode. In glyph mode
// we don't have any value that we can use freely, so instead we represent notdef
// with 3 bytes at 0 instead of two. If what we have are 2 bytes at 0 and one at
// twineCcBegin, that means we are beginning a control sequence instead.

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

const (
	twineRuneEndOfText    rune =  3 // ETX
	twineRuneNotAvailable rune = -1 // only glyph available
)

// Returns the next rune information and the index advance.
// The glyph index is only non-zero if the rune's value is
// twineRuneNotAvailable. You always check the rune first,
// which can also take the twineRuneEndOfText special value.
func (self *Twine) decodeNextAt(index int, inGlyphMode bool) (rune, sfnt.GlyphIndex, int) {
	if index >= len(self.Buffer) { return twineRuneEndOfText, 0, 0 }

	if inGlyphMode {
		glyphIndex := sfnt.GlyphIndex(self.Buffer[index + 1]) << 8
		glyphIndex  = sfnt.GlyphIndex(self.Buffer[index + 0]) | glyphIndex
		if glyphIndex != 0 {
			return twineRuneNotAvailable, glyphIndex, 2
		}

		switch self.Buffer[index + 2] {
		case 0: // true zero, ok (see "[#1]" note)
			return twineRuneNotAvailable, 0, 3
		case uint8(twineCcBegin):
			return rune(twineCcBegin), 0, 3
		}
		panic("invalid twine data")
	} else {
		codePoint, runeLen := utf8.DecodeRune(self.Buffer[index : ])
		if codePoint == utf8.RuneError { panic("invalid rune") }
		return codePoint, 0, runeLen
	}
}

// ---- helpers ----

// used to implement PushEffect, PushMotion, etc.
func (self *Twine) appendKeyWithPayload(cc, key uint8, payload []byte) *Twine {
	self.assertValidPayloadLen(payload...)
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

func (self *Twine) assertValidPayloadLen(payload ...byte) {
	// max payload size assertion
	if len(payload) >= 256 {
		panic( // not ok
			"Maximum payload size on Twine functions is 255, but got " +
			strconv.Itoa(len(payload)) + " bytes instead",
		)
	}
}

// ---- helper functions ----

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

// ---- didn't know where to put this ----

func (self *Renderer) twineStoragePush(value any) {
	self.twineStorage = append(self.twineStorage, value)
}

func (self *Renderer) twineStoragePop() any {
	last := len(self.twineStorage) - 1
	value := self.twineStorage[last]
	self.twineStorage = self.twineStorage[ : last]
	return value
}
