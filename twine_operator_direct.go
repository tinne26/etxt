package etxt

import "strconv"
import "unicode/utf8"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

type twineRewind uint8
const (
	twineNoRewind twineRewind = iota
	twinePreRewind
	twineFullRewind
)

// It's called "direct" because there are two main types of rendering algorithms
// used in etxt: the most general algorithm first measures the text and then
// computes the drawing starting point based on that. Then draws normally. In some
// cases, though, this is not necessary as we already have an horizontal align
// that's compatible with the text direction, and we can start drawing directly.
// (There's also a third case for pure strings where we iterate lines in reverse
// to avoid measuring, but let's ignore that). Twines are tricky as hell and even
// while drawing directly sometimes (preEffects) we have to measure before drawing.
// This means going back and forth between different modes, invoking on line ends/
// starts and changing some random configurations until our brains turn into mush.
// Understanding how different effect types can be stacked and removed and preserved
// or not preserved on measuring or draw mode in a line or while advancing to the
// next one... is pain.
type directTwineOperator struct {
	twine Twine
	index int
	
	order []uint8 // 0 == effect, 1 == preEffect, 2 = motion
	effects []effectOperationData
	preEffects []effectOperationData
	preEffectWidths []fract.Unit // set during pre measuring pass so on draw push the rect is known
	// TODO: motion funcs (or only one motion func? though generalizing wouldn't be such a stretch)

	lineFont *sfnt.Font
	lineDescent fract.Unit
	lineAscent fract.Unit
	lineScaledSize fract.Unit
	inGlyphMode bool
	measuring bool // indicates whether we are measuring or drawing
	notifyLineStartOnRewind bool

	// values that we memorize when switching from draw mode to measure mode
	memoIndex int // index to jump back after processing a pre effect
	memoLineFont *sfnt.Font // memo like preIndex
	memoX fract.Unit
	memoLineDescent fract.Unit // memo like preIndex
	memoLineAscent fract.Unit // memo like preIndex
	memoLineScaledSize fract.Unit // memo like preIndex
	memoPrevGlyphIndex sfnt.GlyphIndex // uint16
	memoNumEffects uint8 // on some rewinds, we need to cut out the new effects
	memoNumPreEffects uint8 // ^ e.g., NotifyLineEnd(), RewindLine()
	memoInGlyphMode bool
}

func (self *directTwineOperator) Initialize(renderer *Renderer, twine Twine) {
	self.twine = twine
	self.RefreshLineMetrics(renderer)
}

// Sometimes we will have both glyphs and runes, other times we will
// only have glyphs, other times we will only have special values given
// through runes, like line breaks, special sequence code (0x1F) or
// end of text (-1). When no rune is available but the glyph index
// is, rune will be -2.
func (self *directTwineOperator) Next() (sfnt.GlyphIndex, rune) {
	if self.index >= len(self.twine.Buffer) { return 0, -1 }

	if self.inGlyphMode {
		glyphIndex := sfnt.GlyphIndex(self.twine.Buffer[self.index + 1]) << 8
		glyphIndex  = sfnt.GlyphIndex(self.twine.Buffer[self.index + 0]) | glyphIndex
		self.index += 2
		if glyphIndex == 0 && self.index < len(self.twine.Buffer) {
			self.index += 1 
			switch self.twine.Buffer[self.index - 1] {
			case 0:
				// true zero, ok (see "[#1]" note on twine.go)
			case uint8(twineCcBegin):
				return 0, rune(twineCcBegin)
			default:
				panic("invalid twine data")
			}
		}
		return glyphIndex, -2
	} else {
		codePoint, runeLen := utf8.DecodeRune(self.twine.Buffer[self.index : ])
		self.index += runeLen
		return 0, codePoint
	}
}

// Process control code. The returned glyph index is the new prev glyph index,
// which can change when breaking glyph pairs or resetting to a previous position.
func (self *directTwineOperator) ProcessCC(renderer *Renderer, target Target, position fract.Point, prevGlyphIndex sfnt.GlyphIndex) (fract.Unit, sfnt.GlyphIndex) {
	controlCode := self.twine.Buffer[self.index]
	switch controlCode {
	case twineCcSwitchGlyphMode: // glyph mode
		self.inGlyphMode = true
		self.index += 1
	case twineCcSwitchStringMode:
		self.inGlyphMode = false
		self.index += 1
	case twineCcPop:
		var preReset bool
		position.X, prevGlyphIndex, preReset = self.Pop(renderer, target, position.X, prevGlyphIndex)
		if !preReset { self.index += 1 }
	case twineCcPopAll:
		var preReset bool
		position.X, prevGlyphIndex, preReset = self.PopAll(renderer, target, position.X, prevGlyphIndex)
		if !preReset { self.index += 1 }
	case twineCcRefreshLineMetrics:
		self.RefreshLineMetrics(renderer)
	case twineCcPushEffect:
		advance, indexOffset := self.PushEffect(renderer, target, self.index + 1, position)
		if advance != 0 { prevGlyphIndex = 0 }
		position.X += advance // don't quantize, next glyph will do that
		self.index += indexOffset + 1
	case twineCcPushPreEffect:
		advance, indexOffset := self.PushPreEffect(renderer, target, self.index + 1, position, prevGlyphIndex)
		if advance != 0 { prevGlyphIndex = 0 }
		position.X += advance // don't quantize, next glyph will do that
		self.index += indexOffset + 1
	case twineCcPushMotion:
		panic("motion effects unimplemented")
	default:
		panic("format code " + strconv.Itoa(int(controlCode)) + " not recognized")
	}

	return position.X, prevGlyphIndex
}

// The returned bool is true whenever the pop triggers a pre reset.
// The glyph index is the new previous glyph index.
func (self *directTwineOperator) Pop(renderer *Renderer, target Target, x fract.Unit, prevGlyphIndex sfnt.GlyphIndex) (fract.Unit, sfnt.GlyphIndex, bool) {
	if len(self.order) == 0 { panic("can't pop on twine: no active directives left") }

	last := len(self.order) - 1
	switch self.order[last] {
	case 0: // post effect
		flags := uint8(TwineTriggerPop) | twineFlagRectOk
		if !self.measuring { flags |= twineFlagDraw }
		advance := self.effects[len(self.effects) - 1].Call(renderer, target, x, flags)
		if advance != 0 { prevGlyphIndex = 0 }
		x += advance
		self.effects = self.effects[0 : len(self.effects) - 1]
	case 1: // pre effect
		// even on pre measuring runs we do want to perform the pop before
		// resetting. the reason is that you can do some shenanigans with
		// the advance to modify the rect, and it's more consistent to have
		// paired push/pops like this. resetting without popping would be
		// possible too, but I think the current behavior is more practical

		flags := uint8(TwineTriggerPop) | twineFlagOnPre | twineFlagRectOk
		if !self.measuring { flags |= twineFlagDraw }

		// call pop with the configured flags
		advance := self.preEffects[len(self.preEffects) - 1].Call(renderer, target, x, flags)
		if advance != 0 { prevGlyphIndex = 0 }
		x += advance
		self.preEffects = self.preEffects[0 : len(self.preEffects) - 1]

		// set pre measuring end position and go back to draw mode when necessary
		if self.measuring {
			// TODO: is this best, or do we want the x stored *before* the pop advance?
			//       this seems somewhat unstable, but may be the more practical choice
			self.preEffectWidths = append(self.preEffectWidths, x - self.memoX)
			if len(self.preEffects) == 0 {
				x = self.restorePreState(renderer, target)
				return x, self.memoPrevGlyphIndex, true
			}
		} else if len(self.preEffects) == 0 {
			self.measuring = false
		}
	case 2: // motion
		panic("motions still unimplemented")
	default:
		panic("broken code")
	}
	
	self.order = self.order[ : last]
	return x, prevGlyphIndex, false
}

// The returned bool is true whenever the pop triggers a pre reset.
// The glyph index is the new previous glyph index.
func (self *directTwineOperator) PopAll(renderer *Renderer, target Target, x fract.Unit, prevGlyphIndex sfnt.GlyphIndex) (fract.Unit, sfnt.GlyphIndex, bool) {
	// If we are on a pre measuring run, we don't always want to pop
	// all effects. If some effects were active before the pre effect
	// push, we want to pop only all other effects and continue on the
	// restored position for drawing the pre effect.
	var preReset bool
	for len(self.order) > 0 {
		x, prevGlyphIndex, preReset = self.Pop(renderer, target, x, prevGlyphIndex)
		if preReset { break }
	}
	return x, prevGlyphIndex, preReset
}

// Precondition: lineBreakX is properly quantized already. lineBreakNth has
// been preincremented (typically drawInternalValues.increaseLineBreakNth()).
// Related methods NotifyLineEnd and NotifyLineStart must be called around this.
func (self *directTwineOperator) AdvanceLine(renderer *Renderer, target Target, position fract.Point, lineBreakX fract.Unit, lineBreakNth int) fract.Point {
	// line advance using operating metrics (only updatable through RefreshLineMetrics())
	if renderer.state.scaledSize == self.lineScaledSize && renderer.state.activeFont == self.lineFont {
		position = renderer.advanceLine(position, lineBreakX, lineBreakNth)
	} else { // when scale and/or font differ, we use the stored twineOperator values (temp set, advance, restore)
		renderer.state.fontSizer.NotifyChange(self.lineFont, &renderer.buffer, self.lineScaledSize)
		tmpFont, tmpSize := renderer.state.activeFont, renderer.state.scaledSize
		renderer.state.activeFont, renderer.state.scaledSize = self.lineFont, self.lineScaledSize
		position = renderer.advanceLine(position, lineBreakX, lineBreakNth)
		renderer.state.activeFont, renderer.state.scaledSize = tmpFont, tmpSize
	}
	return position
}

// The returned bool is true whenever a preReset happens.
// The returned x position is consistent with any necessary reset.
// The returned glyph index is only relevant on resets.
func (self *directTwineOperator) NotifyLineEnd(renderer *Renderer, target Target, x fract.Unit, prevGlyphIndex sfnt.GlyphIndex) (fract.Unit, sfnt.GlyphIndex, bool) {
	postIndex, preIndex := len(self.effects) - 1, len(self.preEffects) - 1
	for i := len(self.order) - 1; i >= 0; i-- {
		switch self.order[i] {
		case 0: // post effect
			flags := uint8(TwineTriggerLineBreak) | twineFlagRectOk
			if !self.measuring { flags |= twineFlagDraw }
			x += self.effects[postIndex].Call(renderer, target, x, flags)
			
			// if we were in measuring mode, since we will rewind,
			// we have to pop the effects after the notification
			if self.measuring {
				self.effects = self.effects[0 : postIndex]
			}
			postIndex -= 1
		case 1: // pre effect
			flags := uint8(TwineTriggerLineBreak) | twineFlagOnPre | twineFlagRectOk
			if !self.measuring { flags |= twineFlagDraw }
			x += self.preEffects[preIndex].Call(renderer, target, x, flags)
			preIndex -= 1

			// if we were in measuring mode, since we will rewind,
			// we have to pop the effects after the notification
			if self.measuring {
				self.preEffects = self.effects[0 : preIndex]
				if len(self.preEffects) == 0 {
					x = self.restorePreState(renderer, target)
					return x, self.memoPrevGlyphIndex, true
				}
			}
		case 2: // motion effect
			panic("motion effects unimplemented")
		default:
			panic("broken code")
		}
	}

	// clear pre effect line end rects if drawing
	if !self.measuring {
		self.preEffectWidths = self.preEffectWidths[ : 0]
	}

	// if on line end pre effects remain unclosed, next line
	// will have to start already measuring
	if len(self.preEffects) > 0 {
		self.notifyLineStartOnRewind = true
		self.measuring = true
	}

	return x, prevGlyphIndex, false
}

// Must be called before NotifyLineStart() if necessary.
func (self *directTwineOperator) NotifyNewBaseline(y fract.Unit) {
	postIndex, preIndex := 0, 0
	for i := 0; i < len(self.order); i++ {
		switch self.order[i] {
		case 0: // post effect
			self.effects[postIndex].origin.Y = y
			postIndex += 1
		case 1: // pre effect
			self.preEffects[preIndex].origin.Y = y
			preIndex += 1
		case 2: // motion
			panic("unimplemented motions")
		default:
			panic("broken code")
		}
	}
}

// Must call NotifyNewBaseline() before this if required.
func (self *directTwineOperator) NotifyLineStart(renderer *Renderer, target Target, x fract.Unit) fract.Unit {
	// special case: don't notify line start if we reached the end
	if self.index >= len(self.twine.Buffer) { return x }

	// regular line start logic
	postIndex, preIndex := 0, 0
	for i := 0; i < len(self.order); i++ {
		switch self.order[i] {
		case 0: // post effect
			flags := uint8(TwineTriggerLineStart)
			if !self.measuring { flags |= twineFlagDraw }
			self.effects[postIndex].origin.X = x
			x += self.effects[postIndex].Call(renderer, target, x, flags)
			postIndex += 1
		case 1: // pre effect
			flags := uint8(TwineTriggerLineStart) | twineFlagOnPre
			if !self.measuring { flags |= twineFlagDraw }
			self.preEffects[preIndex].origin.X = x
			x += self.preEffects[preIndex].Call(renderer, target, x, flags)
			preIndex += 1
		case 2: // motion
			panic("unimplemented motions")
		default:
			panic("broken code")
		}
	}
	return x
}

func (self *directTwineOperator) RefreshLineMetrics(renderer *Renderer) {
	// TODO: unclear if this should also immediately affect effect rects or not,
	//       if only growing, if both growing and becoming smaller, etc. Unclear
	//       if effects should have their own RefreshEffectRect (only affecting
	//       height). It may be actually ok as it is for the moment.
	self.lineDescent = renderer.getOpDescent()
	self.lineAscent = renderer.getOpAscent()
	self.lineScaledSize = renderer.state.scaledSize
	self.lineFont = renderer.state.activeFont
}

func (self *directTwineOperator) PushEffect(renderer *Renderer, target Target, index int, origin fract.Point) (fract.Unit, int) {
	const flags = uint8(TwineTriggerPush) | twineFlagDraw
	op, indexOffset := self.NewEffectOpData(renderer, target, index, origin)
	self.order = append(self.order, 0)
	self.effects = append(self.effects, op)
	return op.Call(renderer, target, origin.X, flags), indexOffset
}

func (self *directTwineOperator) PushPreEffect(renderer *Renderer, target Target, index int, origin fract.Point, prevGlyphIndex sfnt.GlyphIndex) (fract.Unit, int) {
	flags := uint8(TwineTriggerPush) | twineFlagOnPre
	op, indexOffset := self.NewEffectOpData(renderer, target, index, origin)
	self.order = append(self.order, 1)
	self.preEffects = append(self.preEffects, op)
	
	// store state and switch to measuring mode if first pre effect pushed
	if len(self.preEffects) == 1 {
		if self.measuring { panic("broken code") } // TODO: remove once more tested
		self.storePreState(prevGlyphIndex, origin.X)
	}

	// set the draw flag if necessary and invoke
	if !self.measuring { flags |= twineFlagDraw }
	return op.Call(renderer, target, origin.X, flags), indexOffset
}

func (self *directTwineOperator) NewEffectOpData(renderer *Renderer, target Target, index int, origin fract.Point) (effectOperationData, int) {
	var op effectOperationData
	op.key = self.twine.Buffer[index + 0]
	payloadLen := int(self.twine.Buffer[index + 1])
	if payloadLen > 0 {
		start := int(index + 2)
		op.payload = self.twine.Buffer[start : start + payloadLen]
	}
	op.origin = origin
	op.ascent = self.lineAscent
	op.descent = self.lineDescent
	return op, 2 + payloadLen
}

// Notice that memoX and memoPrevGlyphIndex are *not* restored 
// nor returned. They have to be applied/dealt_with manually.
func (self *directTwineOperator) restorePreState(renderer *Renderer, target Target) fract.Unit {
	self.measuring = false

	self.index = self.memoIndex
	self.inGlyphMode = self.memoInGlyphMode
	self.lineDescent = self.memoLineDescent
	self.lineAscent = self.memoLineAscent
	self.lineScaledSize = self.memoLineScaledSize
	self.lineFont = self.memoLineFont
	if self.notifyLineStartOnRewind {
		self.notifyLineStartOnRewind = false
		return self.NotifyLineStart(renderer, target, self.memoX)
	}
	return self.memoX
}

func (self *directTwineOperator) storePreState(prevGlyphIndex sfnt.GlyphIndex, x fract.Unit) {
	self.measuring = true

	self.memoIndex = self.index - 2 // offset required to go back to start call index
	self.memoInGlyphMode = self.inGlyphMode
	self.memoLineDescent = self.lineDescent
	self.memoLineAscent = self.lineAscent
	self.memoLineScaledSize = self.lineScaledSize
	self.memoLineFont = self.lineFont
	
	self.memoX = x
	self.memoPrevGlyphIndex = prevGlyphIndex
}

type effectOperationData struct {
	key uint8
	origin fract.Point
	ascent fract.Unit
	descent fract.Unit
	payload []byte
}

func (self *effectOperationData) Call(renderer *Renderer, target Target, x fract.Unit, flags uint8) fract.Unit {
	var fn TwineEffectFunc
	if self.key > 192 {
		// built in function
		switch TwineEffectKey(self.key) {
		case EffectPushColor : fn = twineEffectPushColor
		case EffectPushFont  : fn = twineEffectPushFont
		case EffectShiftSize : fn = twineEffectShiftSize
		default:
			panic("private TwineEffectFunc #" + strconv.Itoa(int(self.key)) + " is not a defined built-in")
		}
	} else {
		fn = renderer.twineEffectFuncs[self.key]
	}
	
	ymin, ymax := self.origin.Y - self.ascent, self.origin.Y + self.descent
	rect := fract.UnitsToRect(self.origin.X, ymin, x, ymax)
	return fn(renderer, target, TwineEffectArgs{
		Payload: self.payload,
		Rect: rect,
		Origin: self.origin,
		flags: flags,
	})
}

