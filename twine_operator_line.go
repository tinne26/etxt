package etxt

import "strconv"
import "sync/atomic"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// TODO: while measuring, line advances shouldn't notify glyphs cache.
//       In fact, while measuring we don't need to cache so many fields.

// --- twine line operator acquisition ---
// please see twine_operator.go

var reusableTwineLineOperator twineLineOperator
var usingTwineLineOperator uint32 = 0

func getTwineLineOperator() *twineLineOperator {
	if !atomic.CompareAndSwapUint32(&usingTwineLineOperator, 0, 1) {
		var newOperator twineLineOperator
		return &newOperator
	}
	return &reusableTwineLineOperator
}

func releaseTwineLineOperator(twineOperator *twineLineOperator) {
	if twineOperator == &reusableTwineLineOperator {
		atomic.StoreUint32(&usingTwineLineOperator, 0)
	}
}

// --- twine line operator ---

type twineLineOperator struct {
	twine Twine // the twine we are operating on
	index int // current index for twine.Buffer
	inGlyphMode bool // twines can mix glyphs and utf8, so we need to keep track of this
	defaultNewLineX fract.Unit // related to Twine.PushLineRestartMarker
	shiftNewLineX fract.Unit // related to Twine.PushLineRestartMarker
	yCutoff fract.Unit
	effects twineOperatorEffectsList
	spacingPendingAdd *TwineEffectSpacing // dirty hack

	// line values for optimizing line advance operations.
	// if they match the current renderer state values, we can
	// optimize line advance operations. updated when changing lines.
	lineFont *sfnt.Font
	nextLineFont *sfnt.Font
	lineDescent fract.Unit
	nextLineDescent fract.Unit
	lineAscent fract.Unit
	nextLineAscent fract.Unit
	lineScaledSize fract.Unit
	nextLineScaledSize fract.Unit
}

// Must be called before starting to use it. Resets anything left from previous uses, if any.
func (self *twineLineOperator) Initialize(renderer *Renderer, twine Twine, yCutoff fract.Unit) {
	self.twine = twine
	self.defaultNewLineX = 0
	self.shiftNewLineX = 0
	self.yCutoff  = yCutoff
	self.effects.Initialize()
	
	// core resets
	self.index = 0
	self.inGlyphMode = false // twines start in utf8 mode
	
	// get initial line metrics
	self.registerNextLineMetrics(renderer)
	self.newLineMetricsRefresh()
}

func (self *twineLineOperator) SetDefaultNewLineX(x fract.Unit) {
	self.defaultNewLineX = x
}

func (self *twineLineOperator) Ended() bool {
	return self.index >= len(self.twine.Buffer)
}

// The code point can be `twineRuneNotAvailable` if only the glyph is
// known. Otherwise the glyph is always unknown and should be derived
// manually from the rune. See also twineRuneEndOfText and twineCcBegin.
func (self *twineLineOperator) next() (rune, sfnt.GlyphIndex) {
	codePoint, glyphIndex, advance := self.twine.decodeNextAt(self.index, self.inGlyphMode)
	self.index += advance
	return codePoint, glyphIndex
}

// The returned position will only have X affected by quantization, and Y by line advance.
func (self *twineLineOperator) AdvanceLineBreak(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues) {
	position, iv = self.lineBreakAdvance(renderer, position, iv)
	if position.Y > self.yCutoff {
		_, iv = self.measurePopAll(renderer, target, position.X, iv) // no need for the x
	}
	return position, iv
}

// The returned position will only have X affected by quantization, and Y by line advance.
func (self *twineLineOperator) lineBreakAdvance(renderer *Renderer, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues) {
	// line advance using operating metrics (only updatable through RefreshLineMetrics())
	if renderer.state.scaledSize == self.lineScaledSize && renderer.state.activeFont == self.lineFont {
		position = renderer.advanceLine(position, self.defaultNewLineX + self.shiftNewLineX, iv.lineBreakNth)
	} else { // when scale and/or font differ, we use the stored twineOperator values (temp set, advance, restore)
		renderer.state.fontSizer.NotifyChange(self.lineFont, &renderer.buffer, self.lineScaledSize)
		tmpFont, tmpSize := renderer.state.activeFont, renderer.state.scaledSize
		renderer.state.activeFont, renderer.state.scaledSize = self.lineFont, self.lineScaledSize
		position = renderer.advanceLine(position, self.defaultNewLineX + self.shiftNewLineX, iv.lineBreakNth)
		renderer.state.activeFont, renderer.state.scaledSize = tmpFont, tmpSize
	}
	iv.increaseLineBreakNth()
	return position, iv
}

func (self *twineLineOperator) MeasureAndAdvanceLine(renderer *Renderer, target Target, iv drawInternalValues, y fract.Unit) (fract.Unit, drawInternalValues) {
	// safety assertions
	self.effects.AssertAllEffectsActive()
	if self.spacingPendingAdd != nil {
		panic("invalid twineLineOperator usage")
	}

	// measuring time!
	finalPass := true
	var width fract.Unit
	width, iv = self.measureProc(renderer, target, y, iv, finalPass)
	return width, iv
}

// Note: on centered draws, the defaultNewLineX should be manually 
//       reset after each draw or during the placerFn
type placerFn = func(width fract.Unit) fract.Unit
func (self *twineLineOperator) MeasureAndDrawLine(renderer *Renderer, target Target, iv drawInternalValues, y fract.Unit, fn placerFn) (fract.Point, drawInternalValues) {
	// safety assertions
	self.effects.AssertAllEffectsActive()
	if self.spacingPendingAdd != nil {
		panic("invalid twineLineOperator usage")
	}

	// memorize relevant operator state and reconfigure
	memoIndex := self.index
	memoInGlyphMode := self.inGlyphMode
	memoEffectCount := self.effects.ActiveCount()
	memoLineBreakNth := iv.lineBreakNth

	// measuring time!
	finalPass := false
	var width fract.Unit
	width, iv = self.measureProc(renderer, target, y, iv, finalPass)
	iv.lineBreakNth = memoLineBreakNth

	// prepare for drawing by restoring memorized vars and getting the new position
	position := fract.UnitsToPoint(fn(width), y)
	self.index = memoIndex
	self.inGlyphMode = memoInGlyphMode
	
	// fix-everything time!

	// first we recover previously active effects that may have 
	// been popped due to a PopAll()
	for self.effects.ActiveCount() < memoEffectCount {
		effect := self.effects.TryRecallNext()
		if effect == nil { panic("broken code") }
	}
	// next we pop excedent effects that weren't initially
	// active at the start of the line
	for self.effects.ActiveCount() > memoEffectCount {
		effect := self.effects.SoftPop()
		if effect == nil { panic("broken code") }
	}
	// now we call line start in drawing mode for the effects
	// that were already active at the start of the line
	self.effects.Each(func(effect *effectOperationData) {
		const measuring = false
		asc, desc := self.lineAscent, self.lineDescent
		advance := effect.CallLineStart(renderer, target, measuring, &self.twine, asc, desc, position)
		if advance != 0 {
			iv.interruptKerning()
			position.X += renderer.withTextDirSign(advance)
		}
	})

	// drawing time!
loop:
	for {
		codePoint, glyphIndex := self.next()
		switch codePoint {
		case twineRuneEndOfText:
			position.X, iv = self.drawPopAll(renderer, target, position.X, iv)
			break loop
		case '\n':
			position.X, iv = self.drawProcessLineBreak(renderer, target, position.X, iv)
			break loop
		case rune(twineCcBegin):
			position.X, iv = self.drawProcessCC(renderer, target, position.X, position.Y, iv)
		case twineRuneNotAvailable:
			position, iv = self.drawOp(renderer, target, position, glyphIndex, iv)
		default:
			glyphIndex = renderer.getGlyphIndex(renderer.state.activeFont, codePoint)
			position, iv = self.drawOp(renderer, target, position, glyphIndex, iv)
		}
	}
	return self.drawLineBreak(renderer, target, position, iv)
}

func (self *twineLineOperator) measureProc(renderer *Renderer, target Target, y fract.Unit, iv drawInternalValues, finalPass bool) (fract.Unit, drawInternalValues) {
	// basically: while measuring, all pops are soft pops. while drawing, all pops are hard pops.
	// and we probably need some count of something for the "line start effects" (memoEffectCount
	// may suffice)

	// first, call line start for active effects
	var position fract.Point = fract.UnitsToPoint(0, y)
	self.effects.Each(func(effect *effectOperationData) {
		const measuring = true
		asc, desc := self.lineAscent, self.lineDescent
		advance := effect.CallLineStart(renderer, target, measuring, &self.twine, asc, desc, position)
		if advance != 0 {
			iv.interruptKerning()
			position.X += renderer.withTextDirSign(advance)
		}
	})

	// measuring loop
	x := position.X
loop:
	for {
		codePoint, glyphIndex := self.next()
		switch codePoint {
		case twineRuneEndOfText:
			x, iv = self.measurePopAll(renderer, target, x, iv)
			break loop
		case '\n':
			x, iv = self.measureProcessLineBreak(renderer, target, x, iv)
			break loop
		case rune(twineCcBegin):
			x, iv = self.measureProcessCC(renderer, target, x, y, iv, finalPass)
		case twineRuneNotAvailable:
			x, iv = self.measureOp(renderer, x, glyphIndex, iv)
		default:
			glyphIndex = renderer.getGlyphIndex(renderer.state.activeFont, codePoint)
			x, iv = self.measureOp(renderer, x, glyphIndex, iv)
		}
	}

	return renderer.withTextDirSign(x - self.shiftNewLineX), iv
}

func (self *twineLineOperator) measureOp(renderer *Renderer, x fract.Unit, glyphIndex sfnt.GlyphIndex, iv drawInternalValues) (fract.Unit, drawInternalValues) {
	if renderer.state.textDirection == LeftToRight {
		return renderer.advanceGlyphLTR(x, glyphIndex, iv)
	} else { // assume right to left
		return renderer.advanceGlyphRTL(x, glyphIndex, iv)
	}
}

func (self *twineLineOperator) measurePopAll(renderer *Renderer, target Target, x fract.Unit, iv drawInternalValues) (fract.Unit, drawInternalValues) {
	for self.effects.ActiveCount() > 0 {
		x, iv = self.measurePop(renderer, target, x, iv)
	}
	return x, iv
}

func (self *twineLineOperator) measurePop(renderer *Renderer, target Target, x fract.Unit, iv drawInternalValues) (fract.Unit, drawInternalValues) {
	const measuring = true
	effect := self.effects.SoftPop()
	asc, desc := self.lineAscent, self.lineDescent
	advance := effect.CallPop(renderer, target, measuring, &self.twine, asc, desc, x)
	if advance != 0 {
		iv.interruptKerning()
		x += renderer.withTextDirSign(advance)
	}
	return x, iv
}

func (self *twineLineOperator) measureProcessLineBreak(renderer *Renderer, target Target, x fract.Unit, iv drawInternalValues) (fract.Unit, drawInternalValues) {
	const measuring = true
	self.effects.EachReverse(func(effect *effectOperationData) {
		advance := effect.CallLineBreak(renderer, target, measuring, &self.twine, self.lineAscent, self.lineDescent, x)
		if advance != 0 {
			iv.interruptKerning()
			x += renderer.withTextDirSign(advance)
		}
	})
	return x, iv
}

func (self *twineLineOperator) measureProcessCC(renderer *Renderer, target Target, x, y fract.Unit, iv drawInternalValues, finalPass bool) (fract.Unit, drawInternalValues) {
	controlCode := self.twine.Buffer[self.index]
	switch controlCode {
	case twineCcSwitchGlyphMode:
		self.inGlyphMode = true
		self.index += 1
	case twineCcSwitchStringMode:
		self.inGlyphMode = false
		self.index += 1
	case twineCcRefreshLineMetrics:
		if finalPass { self.registerNextLineMetrics(renderer) }
		self.index += 1
	case twineCcPushLineRestartMarker:
		if finalPass { self.shiftNewLineX = x - self.defaultNewLineX }
		self.index += 1
	case twineCcClearLineRestartMarker:
		if finalPass { self.shiftNewLineX = 0 }
		self.index += 1
	case twineCcPushEffectWithSpacing:
		var spacing TwineEffectSpacing
		self.index += 1
		self.index += spacing.parseFromData(self.twine.Buffer[self.index : ])
		nextCc := self.twine.Buffer[self.index]
		if nextCc != twineCcPushSinglePassEffect && nextCc != twineCcPushDoublePassEffect {
			panic("invalid PushEffectWithSpacing twine contents")
		}
		self.spacingPendingAdd = &spacing
		return self.measureProcessCC(renderer, target, x, y, iv, finalPass)
	case twineCcPop:
		x, iv = self.measurePop(renderer, target, x, iv)
		self.index += 1
	case twineCcPopAll:
		x, iv = self.measurePopAll(renderer, target, x, iv)
		self.index += 1
	case twineCcPushSinglePassEffect, twineCcPushDoublePassEffect:
		const measuring = true
		if controlCode == twineCcPushSinglePassEffect {
			self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, SinglePass, measuring)
		} else {
			self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, DoublePass, measuring)
		}
		
		position := fract.UnitsToPoint(x, y)
		asc, desc := self.lineAscent, self.lineDescent
		advance := self.effects.Head().CallPush(renderer, target, measuring, &self.twine, asc, desc, position)
		if advance != 0 {
			iv.interruptKerning()
			x += renderer.withTextDirSign(advance)
		}
	case twineCcPushMotion:
		panic("motion effects unimplemented")
	default:
		panic("format code " + strconv.Itoa(int(controlCode)) + " not recognized")
	}

	return x, iv
}

// ---- draw operations ----

func (self *twineLineOperator) drawPopAll(renderer *Renderer, target Target, x fract.Unit, iv drawInternalValues) (fract.Unit, drawInternalValues) {
	for self.effects.ActiveCount() > 0 {
		x, iv = self.drawPop(renderer, target, x, iv)
	}
	return x, iv
}

func (self *twineLineOperator) drawPop(renderer *Renderer, target Target, x fract.Unit, iv drawInternalValues) (fract.Unit, drawInternalValues) {
	const measuring = false
	effect := self.effects.Head()
	asc, desc := self.lineAscent, self.lineDescent
	advance := effect.CallPop(renderer, target, measuring, &self.twine, asc, desc, x)
	if advance != 0 {
		iv.interruptKerning()
		x += renderer.withTextDirSign(advance)
	}
	self.effects.HardPop()
	return x, iv
}

func (self *twineLineOperator) drawProcessLineBreak(renderer *Renderer, target Target, x fract.Unit, iv drawInternalValues) (fract.Unit, drawInternalValues) {
	const measuring = false
	self.effects.EachReverse(func(effect *effectOperationData) {
		advance := effect.CallLineBreak(renderer, target, measuring, &self.twine, self.lineAscent, self.lineDescent, x)
		if advance != 0 {
			iv.interruptKerning()
			x += renderer.withTextDirSign(advance)
		}
	})
	return x, iv
}

func (self *twineLineOperator) drawProcessCC(renderer *Renderer, target Target, x, y fract.Unit, iv drawInternalValues) (fract.Unit, drawInternalValues) {
	controlCode := self.twine.Buffer[self.index]
	switch controlCode {
	case twineCcSwitchGlyphMode:
		self.inGlyphMode = true
		self.index += 1
	case twineCcSwitchStringMode:
		self.inGlyphMode = false
		self.index += 1
	case twineCcRefreshLineMetrics:
		self.registerNextLineMetrics(renderer)
		self.index += 1
	case twineCcPushLineRestartMarker:		
		self.shiftNewLineX = x - self.defaultNewLineX
		self.index += 1
	case twineCcClearLineRestartMarker:
		self.shiftNewLineX = 0
		self.index += 1
	case twineCcPushEffectWithSpacing:
		var spacing TwineEffectSpacing
		self.index += 1
		self.index += spacing.parseFromData(self.twine.Buffer[self.index : ])
		nextCc := self.twine.Buffer[self.index]
		if nextCc != twineCcPushSinglePassEffect && nextCc != twineCcPushDoublePassEffect {
			panic("invalid PushEffectWithSpacing twine contents")
		}
		self.spacingPendingAdd = &spacing
		return self.drawProcessCC(renderer, target, x, y, iv)
	case twineCcPop:
		x, iv = self.drawPop(renderer, target, x, iv)
		self.index += 1
	case twineCcPopAll:
		x, iv = self.drawPopAll(renderer, target, x, iv)
		self.index += 1
	case twineCcPushSinglePassEffect, twineCcPushDoublePassEffect:
		const measuring = false
		if controlCode == twineCcPushSinglePassEffect {
			self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, SinglePass, measuring)
		} else {
			self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, DoublePass, measuring)
		}
		
		position := fract.UnitsToPoint(x, y)
		asc, desc := self.lineAscent, self.lineDescent
		advance := self.effects.Head().CallPush(renderer, target, measuring, &self.twine, asc, desc, position)
		if advance != 0 {
			iv.interruptKerning()
			x += renderer.withTextDirSign(advance)
		}
	case twineCcPushMotion:
		panic("motion effects unimplemented")
	default:
		panic("format code " + strconv.Itoa(int(controlCode)) + " not recognized")
	}

	return x, iv
}

func (self *twineLineOperator) drawOp(renderer *Renderer, target Target, position fract.Point, glyphIndex sfnt.GlyphIndex, iv drawInternalValues) (fract.Point, drawInternalValues) {
	if renderer.state.textDirection == LeftToRight {
		return renderer.drawGlyphLTR(target, position, glyphIndex, iv)
	} else { // assume right to left
		return renderer.drawGlyphRTL(target, position, glyphIndex, iv)
	}
}

// The returned position will only have X affected by quantization, and Y by line advance.
func (self *twineLineOperator) drawLineBreak(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues) {
	position, iv = self.lineBreakAdvance(renderer, position, iv) // will only affect the position's Y
	if position.Y > self.yCutoff {
		_, iv = self.drawPopAll(renderer, target, position.X, iv) // no need for the position
	}
	return position, iv
}

// ---- internal helpers ---- 

func (self *twineLineOperator) registerNextLineMetrics(renderer *Renderer) {
	self.nextLineDescent = renderer.getOpDescent()
	self.nextLineAscent = renderer.getOpAscent()
	self.nextLineScaledSize = renderer.state.scaledSize
	self.nextLineFont = renderer.state.activeFont
}

func (self *twineLineOperator) newLineMetricsRefresh() {
	self.lineFont = self.nextLineFont
	self.lineDescent = self.nextLineDescent
	self.lineAscent = self.nextLineAscent
	self.lineScaledSize = self.nextLineScaledSize
}

func (self *twineLineOperator) appendNewEffectOpDataWithKeyAt(index int, effectMode TwineEffectMode, measuring bool) int {
	// note: the data structure is [twineCcBegin, effectModeCc, key, payloadLen, payload...]
	key := self.twine.Buffer[index]
	payloadLen := self.twine.Buffer[index + 1]

	var effect *effectOperationData
	if !measuring { effect = self.effects.TryRecallNext() }
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
		opData.spacing = self.spacingPendingAdd
		self.spacingPendingAdd = nil

		// push op data
		self.effects.Push(opData)
	}

	// clear spacePendingAdd and return new twine buffer index
	self.spacingPendingAdd = nil
	return index + int(payloadLen) + 2
}
