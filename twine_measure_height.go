package etxt

import "strconv"
import "sync/atomic"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

var reusableTwineHeightSizer twineHeightSizer
var usingTwineHeightSizer uint32 = 0

func getTwineHeightSizer() *twineHeightSizer {
	if !atomic.CompareAndSwapUint32(&usingTwineHeightSizer, 0, 1) {
		var newSizer twineHeightSizer
		return &newSizer
	}
	return &reusableTwineHeightSizer
}

func releaseTwineHeightSizer(twineSizer *twineHeightSizer) {
	if twineSizer == &reusableTwineHeightSizer {
		atomic.StoreUint32(&usingTwineHeightSizer, 0)
	}
}

// TODO: line start not called anywhere, jeez

// note: maybe it would be clever to try to measure without line height
//       changes first, and only if that fails fall back to full measuring.
//       to me, it feels like most of the time we won't be using line
//       height refreshes anyway, so measuring would be wasteful half of
//       the time.
type twineHeightSizer struct {
	twine Twine
	index int
	inGlyphMode bool
	lineBreakNth int
	effects twineOperatorEffectsList

	lineFont *sfnt.Font
	nextLineFont *sfnt.Font
	lineScaledSize fract.Unit
	nextLineScaledSize fract.Unit
	lineHeight fract.Unit
	nextLineHeight fract.Unit
	lineAscent fract.Unit
	nextLineAscent fract.Unit
	lineDescent fract.Unit
	nextLineDescent fract.Unit
	// note: we can't cache lineAdvance, that 
	//       depends on lineBreakNth and so on
}

func (self *twineHeightSizer) Initialize(renderer *Renderer, twine Twine) {
	self.twine = twine
	self.effects.Initialize()
	
	// core resets
	self.index = 0
	self.inGlyphMode = false // twines start in utf8 mode
	
	// get initial line metrics
	self.registerNextLineMetrics(renderer)
	self.newLineMetricsRefresh()
}

func (self *twineHeightSizer) Measure(renderer *Renderer, target Target) fract.Unit {
	height := fract.Unit(0)
	lineBreaksOnly := true

	for {
		codePoint, _, bytesAdvance := self.twine.decodeNextAt(self.index, self.inGlyphMode)
		self.index += bytesAdvance

		switch codePoint {
		case twineRuneEndOfText:
			return height.QuantizeUp(fract.Unit(renderer.state.vertQuantization))
		case '\n':
			height, lineBreaksOnly = self.lineBreak(renderer, target, height, lineBreaksOnly)
		case rune(twineCcBegin):
			advance := self.processCC(renderer, target)
			if lineBreaksOnly && advance != 0 {
				lineBreaksOnly = false
				height += self.lineHeight
			}
		default:
			self.lineBreakNth = 0
			if lineBreaksOnly {
				lineBreaksOnly = false
				height += self.lineHeight
			}
		}
	}

	panic("unreachable")
}

func (self *twineHeightSizer) processCC(renderer *Renderer, target Target) fract.Unit {
	controlCode := self.twine.Buffer[self.index]
	switch controlCode {
	case twineCcSwitchGlyphMode: // glyph mode
		self.inGlyphMode = true
		self.index += 1
	case twineCcSwitchStringMode:
		self.inGlyphMode = false
		self.index += 1
	case twineCcRefreshLineMetrics:
		self.registerNextLineMetrics(renderer)
		self.index += 1
	case twineCcPushLineRestartMarker, twineCcClearLineRestartMarker:
		// these are irrelevant for height
		self.index += 1
	case twineCcPushEffectWithSpacing:
		var spacing TwineEffectSpacing
		self.index += 1
		self.index += spacing.parseFromData(self.twine.Buffer[self.index : ])
		nextCc := self.twine.Buffer[self.index]
		if nextCc != twineCcPushSinglePassEffect && nextCc != twineCcPushDoublePassEffect {
			panic("invalid PushEffectWithSpacing twine contents")
		}
		return self.processCC(renderer, target)
	case twineCcPop:
		advance := self.pop(renderer, target)
		if advance != 0 { self.lineBreakNth = -1 }
		self.index += 1
		return advance
	case twineCcPopAll:
		advance := self.popAll(renderer, target)
		if advance != 0 { self.lineBreakNth = -1 }
		self.index += 1
		return advance
	case twineCcPushSinglePassEffect:
		self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, SinglePass)
		asc, desc := self.lineAscent, self.lineDescent
		pt := fract.Point{0, 0}
		advance := self.effects.Head().CallPush(renderer, target, true, &self.twine, asc, desc, pt)
		if advance != 0 { self.lineBreakNth = -1 }
		return advance
	case twineCcPushDoublePassEffect:
		self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, DoublePass)
		asc, desc := self.lineAscent, self.lineDescent
		pt := fract.Point{0, 0}
		advance := self.effects.Head().CallPush(renderer, target, true, &self.twine, asc, desc, pt)
		if advance != 0 { self.lineBreakNth = -1 }
		return advance
	case twineCcPushMotion:
		panic("motion effects unimplemented")
	default:
		panic("format code " + strconv.Itoa(int(controlCode)) + " not recognized")
	}

	return 0
}

func (self *twineHeightSizer) lineBreak(renderer *Renderer, target Target, height fract.Unit, lineBreaksOnly bool) (fract.Unit, bool) {
	vertQuant := fract.Unit(renderer.state.vertQuantization)
	self.effects.EachReverse(func(effect *effectOperationData) {
		const measuring = true
		asc, desc := self.lineAscent, self.lineDescent
		advance := effect.CallLineBreak(renderer, target, measuring, &self.twine, asc, desc, 0)
		// note: using 0 here ^ is certainly debatable.
		if lineBreaksOnly && advance != 0 {
			lineBreaksOnly = false
			height += self.lineHeight
		}
	})

	// advance line
	height = (height + self.getLineAdvanceUnits(renderer)).QuantizeUp(vertQuant)
	self.lineBreakNth += 1

	// call line starts
	y := height
	self.effects.Each(func(effect *effectOperationData) {
		const measuring = true
		asc, desc := self.lineAscent, self.lineDescent
		advance := effect.CallLineStart(renderer, target, measuring, &self.twine, asc, desc, fract.Point{0, y})
		// note: using 0 here ^ is, again, debatable.
		if lineBreaksOnly && advance != 0 {
			lineBreaksOnly = false
			height += self.lineHeight
		}
	})

	return height, lineBreaksOnly
}

// ---- helpers ----

func (self *twineHeightSizer) appendNewEffectOpDataWithKeyAt(index int, effectMode TwineEffectMode) int {
	key := self.twine.Buffer[index]
	payloadLen := self.twine.Buffer[index + 1]

	effect := self.effects.TryRecallNext()
	if effect == nil { // manual push required
		// create op data
		var opData effectOperationData
		opData.key = key
		payloadLen := self.twine.Buffer[index + 1]
		if payloadLen > 0 {
			opData.payloadStartIndex = uint32(index) + 2
			opData.payloadEndIndex = opData.payloadStartIndex + uint32(payloadLen)
		}
		opData.mode = effectMode

		// push op data
		self.effects.Push(opData)
	}
	return index + int(payloadLen) + 2
}

func (self *twineHeightSizer) registerNextLineMetrics(renderer *Renderer) {
	self.nextLineAscent  = renderer.getOpAscent()
	self.nextLineDescent = renderer.getOpDescent()
	self.nextLineHeight  = renderer.getOpLineHeight()
	self.nextLineFont    = renderer.state.activeFont
	self.nextLineScaledSize = renderer.state.scaledSize
}

func (self *twineHeightSizer) newLineMetricsRefresh() {
	self.lineFont    = self.nextLineFont
	self.lineHeight  = self.nextLineHeight
	self.lineAscent  = self.nextLineAscent
	self.lineDescent = self.nextLineDescent
	self.lineScaledSize = self.nextLineScaledSize
}

func (self *twineHeightSizer) pop(renderer *Renderer, target Target) fract.Unit {
	const measuring = true
	asc, desc := self.lineAscent, self.lineDescent
	advance := self.effects.Head().CallPop(renderer, target, measuring, &self.twine, asc, desc, 0)
	self.effects.HardPop()
	if advance != 0 { self.lineBreakNth = -1 }
	return advance
}

func (self *twineHeightSizer) popAll(renderer *Renderer, target Target) fract.Unit {
	var advance fract.Unit
	for self.effects.ActiveCount() > 0 {
		advance += self.pop(renderer, target)
	}
	return advance
}

func (self *twineHeightSizer) getLineAdvanceUnits(renderer *Renderer) fract.Unit {
	// line advance using operating metrics (only updatable through RefreshLineMetrics())
	var advance fract.Unit
	if renderer.state.scaledSize == self.lineScaledSize && renderer.state.activeFont == self.lineFont {
		advance = renderer.getOpLineAdvance(self.lineBreakNth)
	} else { // when scale and/or font differ, we use the stored twineOperator values (temp set, advance, restore)
		renderer.state.fontSizer.NotifyChange(self.lineFont, &renderer.buffer, self.lineScaledSize)
		tmpFont, tmpSize := renderer.state.activeFont, renderer.state.scaledSize
		renderer.state.activeFont, renderer.state.scaledSize = self.lineFont, self.lineScaledSize
		advance = renderer.getOpLineAdvance(self.lineBreakNth)
		renderer.state.activeFont, renderer.state.scaledSize = tmpFont, tmpSize
	}
	return advance
}
