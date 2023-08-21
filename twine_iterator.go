package etxt

import "strconv"
import "unicode/utf8"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

type ltrTwineIterator struct {
	index int
	inGlyphMode bool
}

// Sometimes we will have both glyphs and runes, other times we will
// only have glyphs, other times we will only have special values given
// through runes, like line breaks, special sequence code (0x1F) or
// end of text (-1). When no rune is available but the glyph index
// is, rune will be -2.
func (self *ltrTwineIterator) Next(twine Twine) (sfnt.GlyphIndex, rune) {
	if self.inGlyphMode {
		glyphIndex := sfnt.GlyphIndex(twine.Buffer[self.index + 1]) << 8
		glyphIndex  = sfnt.GlyphIndex(twine.Buffer[self.index + 0]) | glyphIndex
		self.index += 2
		if glyphIndex == 0 && self.index < len(twine.Buffer) {
			self.index += 1 
			switch twine.Buffer[self.index - 1] {
			case 0:
				// true zero, ok
			case uint8(twineCcBegin):
				return 0, rune(twineCcBegin)
			default:
				panic("invalid twine data")
			}
		}
		return glyphIndex, -2
	} else {
		if self.index >= len(twine.Buffer) { return 0, -1 }
		codePoint, runeLen := utf8.DecodeRune(twine.Buffer[self.index : ])
		self.index += runeLen
		return 0, codePoint
	}
}

func (self *ltrTwineIterator) ProcessCC(target Target, renderer *Renderer, operator twineOperator, position fract.Point, iv drawInternalValues) (twineOperator, fract.Point, drawInternalValues) {
	controlCode := operator.twine.Buffer[self.index]
	switch controlCode {
	case twineCcSwitchGlyphMode: // glyph mode
		self.inGlyphMode = true
		self.index += 1
	case twineCcSwitchStringMode:
		self.inGlyphMode = false
		self.index += 1
	case twineCcPop:
		position.X = operator.Pop(renderer, target, position.X)
		self.index += 1
	case twineCcPopAll:
		position.X = operator.PopAll(renderer, target, position.X)
		self.index += 1
	case twineCcRefreshLineMetrics:
		// ... (implement later)
		panic("refresh line metrics unimplemented")
	case twineCcPushEffect:
		advance, indexOffset := operator.PushEffect(renderer, target, self.index + 1, position)
		if advance != 0 {
			position.X += advance // don't quantize, next glyph will do that
			iv.prevGlyphIndex = 0
			iv.lineBreakNth = -1
		}
		self.index += indexOffset + 1
	case twineCcPushPreEffect:
		panic("push pre effect unimplemented")
	case twineCcPushMotion:
		panic("push motion unimplemented")
	default:
		panic("format code " + strconv.Itoa(int(controlCode)) + " not recognized")
	}

	return operator, position, iv
}

type twineOperator struct {
	twine Twine
	order []uint8 // 0 == effect, 1 == preEffect, 2 = motion
	effects []effectOperationData
	preEffects []effectOperationData
	lineDescent fract.Unit
	lineAscent fract.Unit
	lineScaledSize fract.Unit
	lineFont *sfnt.Font

	// TODO: motion funcs (or only one motion func)
}

func newTwineOperator(renderer *Renderer, twine Twine) twineOperator {
	var operator twineOperator
	operator.twine = twine
	operator.RefreshLineMetrics(renderer)
	return operator
}

func (self *twineOperator) Pop(renderer *Renderer, target Target, x fract.Unit) fract.Unit {
	if len(self.order) == 0 { panic("can't pop on twine: no active directives left") }

	last := len(self.order) - 1
	switch self.order[last] {
	case 0: // post effect
		const flags = TwineTriggerPop | TwineFlagDraw
		x += self.effects[len(self.effects) - 1].Call(renderer, target, x, flags)
		self.effects = self.effects[0 : len(self.effects)]
	case 1: // pre effect
		const flags = TwineTriggerPop | TwineFlagDraw | TwineFlagPre
		x += self.preEffects[len(self.preEffects) - 1].Call(renderer, target, x, flags)
		self.preEffects = self.preEffects[0 : len(self.preEffects)]
	case 2: // motion
		panic("unimplemented")
	default:
		panic("broken code")
	}
	
	self.order = self.order[ : last]
	return x
}

func (self *twineOperator) PopAll(renderer *Renderer, target Target, x fract.Unit) fract.Unit {
	for len(self.order) > 0 {
		x = self.Pop(renderer, target, x)
	}
	return x
}

func (self *twineOperator) LineBreak() {
	// ... pain
}

func (self *twineOperator) RefreshLineMetrics(renderer *Renderer) {
	self.lineDescent = renderer.getOpDescent()
	self.lineAscent = renderer.getOpAscent()
	self.lineScaledSize = renderer.state.scaledSize
	self.lineFont = renderer.state.activeFont
}

func (self *twineOperator) PushEffect(renderer *Renderer, target Target, index int, origin fract.Point) (fract.Unit, int) {
	const flags = TwineTriggerPush | TwineFlagDraw
	op, indexOffset := self.NewEffectOpData(renderer, target, index, origin)
	self.order = append(self.order, 0)
	self.effects = append(self.effects, op)
	return op.Call(renderer, target, origin.X, flags), indexOffset
}

func (self *twineOperator) PushPreEffect(renderer *Renderer, target Target, index int, origin fract.Point) (fract.Unit, int) {
	const flags = TwineTriggerPush | TwineFlagDraw | TwineFlagPre
	op, indexOffset := self.NewEffectOpData(renderer, target, index, origin)
	self.order = append(self.order, 1)
	self.preEffects = append(self.preEffects, op)
	return op.Call(renderer, target, origin.X, flags), indexOffset
}

type effectOperationData struct {
	key uint8
	origin fract.Point
	ascent fract.Unit
	descent fract.Unit
	payload []byte
}

func (self *twineOperator) NewEffectOpData(renderer *Renderer, target Target, index int, origin fract.Point) (effectOperationData, int) {
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

func (self *effectOperationData) Call(renderer *Renderer, target Target, x fract.Unit, flags TwineEffectFlags) fract.Unit {
	var fn TwineEffectFunc
	if self.key > 192 {
		// built in function
		switch TwineEffectKey(self.key) {
		case EffectPushColor : fn = twineEffectPushColor
		case EffectPushFont  : fn = twineEffectPushFont
		default:
			panic("private TwineEffectFunc #" + strconv.Itoa(int(self.key)) + " not found")
		}
	} else {
		fn = renderer.twineEffectFuncs[self.key]
	}
	
	ymin, ymax := self.origin.Y - self.ascent, self.origin.Y + self.descent
	rect := fract.UnitsToRect(self.origin.X, ymin, x, ymax)
	return fn(renderer, self.payload, flags, target, self.origin, rect)
}
