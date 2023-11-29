package etxt

import "strconv"
import "sync/atomic"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// double pass reset mode constants
type dpResetMode uint8
const (
	dpResetNone      dpResetMode = 0
	dpResetPush      dpResetMode = 1
	dpResetLineStart dpResetMode = 2
)

// --- twine operator acquisition ---
// same strategy as etxt/font.getSfntBuffer()
var reusableTwineOperator twineOperator
var usingTwineOperator uint32 = 0

// notice: this will always fail if we panic but there's a top-level
// recovery, leading to a permanently skipped releaseTwineOperator.
// it's not the worst thing ever, but it's not nice.
func getTwineOperator() *twineOperator {
	if !atomic.CompareAndSwapUint32(&usingTwineOperator, 0, 1) {
		var newOperator twineOperator
		return &newOperator
	}
	return &reusableTwineOperator
}

func releaseTwineOperator(twineOperator *twineOperator) {
	if twineOperator == &reusableTwineOperator {
		atomic.StoreUint32(&usingTwineOperator, 0)
	}
}

// --- twine operator ---

type twineOperator struct {
	// TODO:
	// - struct field reordering: do late or never. code is already too complex

	twine Twine // the twine we are operating on
	index int // current index for twine.Buffer
	inGlyphMode bool // twines can mix glyphs and utf8, so we need to keep track of this
	performDrawPass bool // whether the operator must draw or only measure
	onMeasuringPass bool // whether we are on a measuring or drawing pass
	defaultNewLineX fract.Unit // related to Twine.PushLineRestartMarker
	fixedNewLineX fract.Unit // related to Twine.PushLineRestartMarker
	yCutoff fract.Unit
	effects []effectOperationData
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

	// - double pass reset values -
	doublePassResetMode dpResetMode
	doublePassResetInGlyphMode bool
	doublePassResetEffectIndex uint16 // self.effects index for the first DoublePass effect (the reset point)
	doublePassResetX fract.Unit
	doublePassResetLineBreakNth int32
	doublePassResetPrevGlyphIndex sfnt.GlyphIndex // uint16, 2 bytes
	doublePassResetIndex int // index to jump back after first measuring pass.
	                         // the index is placed after the effect's CC and payload
}

// Must be called before starting to use it. Resets anything left from previous uses, if any.
func (self *twineOperator) Initialize(renderer *Renderer, twine Twine, mustDraw bool, newLineX, yCutoff fract.Unit) {
	self.twine = twine
	self.performDrawPass = mustDraw
	self.onMeasuringPass = !mustDraw
	self.defaultNewLineX = newLineX
	self.fixedNewLineX = newLineX
	self.yCutoff  = yCutoff
	if cap(self.effects) < 8 {
		self.effects = make([]effectOperationData, 0, 8)
	} else {
		self.effects = self.effects[ : 0]
	}
	
	// core resets
	self.index = 0
	self.inGlyphMode = false // twines start in utf8 mode
	self.doublePassResetMode = dpResetNone
	
	// get initial line metrics
	self.registerNextLineMetrics(renderer)
	self.newLineMetricsRefresh()
}

// Get the next code point and glyph index. The code point can be the
// special value `twineRuneNotAvailable` if only the glyph is known.
// Otherwise the glyph is always unknown and should be derived manually
// from the rune.
// The only other possible special value is twineRuneEndOfText.
// Important: if the rune is twineCcBegin, ProcessCC must be invoked
// afterwards.
func (self *twineOperator) Next() (rune, sfnt.GlyphIndex) {
	codePoint, glyphIndex, advance := self.twine.decodeNextAt(self.index, self.inGlyphMode)
	self.index += advance
	return codePoint, glyphIndex
}

func (self *twineOperator) OperateLTR(renderer *Renderer, target Target, position fract.Point, glyphIndex sfnt.GlyphIndex, iv drawInternalValues) (fract.Point, drawInternalValues) {
	if self.onMeasuringPass {
		return renderer.advanceGlyphLTR(target, position, glyphIndex, iv)
	} else { // drawing
		return renderer.drawGlyphLTR(target, position, glyphIndex, iv)
	}
}

func (self *twineOperator) LineBreak(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues) {
	// notify line end, which could trigger a double pass reset
	var dpReset bool
	position, iv, dpReset = self.notifyLineBreak(renderer, target, position, iv)
	if dpReset { return position, iv }

	// start setting everything up for next line
	iv.increaseLineBreakNth()
	
	// line advance using operating metrics (only updatable through RefreshLineMetrics())
	if renderer.state.scaledSize == self.lineScaledSize && renderer.state.activeFont == self.lineFont {
		position = renderer.advanceLine(position, self.fixedNewLineX, iv.lineBreakNth)
	} else { // when scale and/or font differ, we use the stored twineOperator values (temp set, advance, restore)
		renderer.state.fontSizer.NotifyChange(self.lineFont, &renderer.buffer, self.lineScaledSize)
		tmpFont, tmpSize := renderer.state.activeFont, renderer.state.scaledSize
		renderer.state.activeFont, renderer.state.scaledSize = self.lineFont, self.lineScaledSize
		position = renderer.advanceLine(position, self.fixedNewLineX, iv.lineBreakNth)
		renderer.state.activeFont, renderer.state.scaledSize = tmpFont, tmpSize
	}

	// check y cutoff to terminate early and pop every remaining effect in that case
	if position.Y > self.yCutoff {
		position, iv, _ = self.PopAll(renderer, target, position, iv)
		return position, iv
	}

	// update/invoke effects for the new line
	if !self.performDrawPass || self.doublePassResetMode == dpResetNone {
		// easy case: we don't have to draw, so it's all just measuring
		// (effects changing font or font size can still affect metrics).
		// just stay on measuring pass and trigger a LineStart for everyone
		for i, _ := range self.effects {
			self.effects[i].origin = position
			advance := self.effects[i].CallLineStart(renderer, target, self, position)
			if advance != 0 {
				iv.interruptKerning()
				position.X += renderer.withTextDirSign(advance)
			}
		}
	} else {
		// super annoying tricky case! if we still have double pass effects accumulated,
		// we will need to memorize a new reset point and go into measure mode again...
		// but only once we hit the first double pass effect. single pass effects wrapping
		// from further back can already be "drawn" when hit.
		// 
		// the loop is similar to the previous branch, but with some extra logic (just
		// read the comments)
		if self.onMeasuringPass { panic("assertion") }
		for i, _ := range self.effects {
			// if we find a double pass effect while still drawing,
			// it's time to switch to measure mode and set our reset point
			if !self.onMeasuringPass && self.effects[i].mode == DoublePass {
				self.onMeasuringPass = true
				self.storeDoublePassResetData(position, iv, dpResetLineStart)
			}

			// regular loop logic (update effect positions, call them with TwineTriggerStartLine)
			self.effects[i].origin = position
			advance := self.effects[i].CallLineStart(renderer, target, self, position)
			if advance != 0 {
				iv.interruptKerning()
				position.X += renderer.withTextDirSign(advance)
			}
		}
		if !self.onMeasuringPass { panic("assertion") }
	}
	
	return position, iv
}

// Must be called when twineCcBegin is encountered on twineOperator.Next().
func (self *twineOperator) ProcessCC(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues) {
	var dpReset bool

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
	case twineCcPushLineRestartMarker:
		self.fixedNewLineX = position.X
		self.index += 1
	case twineCcClearLineRestartMarker:
		self.fixedNewLineX = self.defaultNewLineX
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
		return self.ProcessCC(renderer, target, position, iv)
	case twineCcPop:
		position, iv, dpReset = self.pop(renderer, target, position, iv)
		if !dpReset { self.index += 1 } // otherwise the index is set on pop >> restorePreState
	case twineCcPopAll:
		position, iv, dpReset = self.PopAll(renderer, target, position, iv)
		if !dpReset { self.index += 1 }
	case twineCcPushSinglePassEffect:
		self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, SinglePass)
		advance := self.effects[len(self.effects) - 1].CallPush(renderer, target, self, position)
		if advance > 0 {
			iv.interruptKerning()
			position.X += renderer.withTextDirSign(advance)
		}
	case twineCcPushDoublePassEffect:
		self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, DoublePass)
		self.notifyAddedDoublePassEffect(position, iv)
		advance := self.effects[len(self.effects) - 1].CallPush(renderer, target, self, position)
		if advance > 0 {
			iv.interruptKerning()
			position.X += renderer.withTextDirSign(advance)
		}
	case twineCcPushMotion:
		panic("motion effects unimplemented")
	default:
		panic("format code " + strconv.Itoa(int(controlCode)) + " not recognized")
	}

	return position, iv
}

func (self *twineOperator) PopAll(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues, bool) {
	var dpReset bool
	for len(self.effects) > 0 {
		position, iv, dpReset = self.pop(renderer, target, position, iv)
		if dpReset { return position, iv, true }
	}
	return position, iv, false
}

// --- internal core functions ---

// This should only be called on init or new lines.
func (self *twineOperator) newLineMetricsRefresh() {
	self.lineFont = self.nextLineFont
	self.lineDescent = self.nextLineDescent
	self.lineAscent = self.nextLineAscent
	self.lineScaledSize = self.nextLineScaledSize
}

func (self *twineOperator) registerNextLineMetrics(renderer *Renderer) {
	self.nextLineDescent = renderer.getOpDescent()
	self.nextLineAscent = renderer.getOpAscent()
	self.nextLineScaledSize = renderer.state.scaledSize
	self.nextLineFont = renderer.state.activeFont
}

func (self *twineOperator) pop(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues, bool) {
	effectIndex := len(self.effects) - 1
	if effectIndex < 0 { panic("can't pop on twine: no active effects left") }

	// call pop on the effect
	advance := self.effects[effectIndex].CallPop(renderer, target, self, position.X)
	if advance > 0 {
		iv.interruptKerning()
		position.X += renderer.withTextDirSign(advance)
	}

	// either reset or remove the effect from the effects slice
	if self.doublePassResetMode != dpResetNone && effectIndex == int(self.doublePassResetEffectIndex) {
		position, iv = self.resetForSecondPass(renderer, target, position, iv)
		return position, iv, true
	} else { // simple pop case
		self.effects = self.effects[ : len(self.effects) - 1]
		return position, iv, false
	}
}

// Appends a new effect op data value and returns the new index (first position after the payload).
// One must still invoke CallPush or CallLineStart on the effect afterwards.
func (self *twineOperator) appendNewEffectOpDataWithKeyAt(index int, effectMode TwineEffectMode) int {
	// note: the data structure is [twineCcBegin, effectModeCc, key, payloadLen, payload...]
	var opData effectOperationData
	opData.key = self.twine.Buffer[index]
	payloadLen := self.twine.Buffer[index + 1]
	if payloadLen > 0 {
		opData.payloadStartIndex = uint32(index) + 2
		opData.payloadEndIndex = opData.payloadStartIndex + uint32(payloadLen)
	}
	opData.mode = effectMode
	if self.spacingPendingAdd != nil {
		opData.spacing = self.spacingPendingAdd
		self.spacingPendingAdd = nil
	}
	self.effects = append(self.effects, opData)
	return index + int(payloadLen) + 2
}

func (self *twineOperator) notifyLineBreak(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues, bool) {
	// we have to call LineBreak on most effects, but there's the possibility of a double
	// pass reset midway, before all effects have line end called on them. so, first,
	// figure out if a reset will be necessary and at which point it would happen
	resetBreakIndex := -1
	if (self.doublePassResetMode != dpResetNone && self.onMeasuringPass && self.performDrawPass) {
		for i := 0; i < len(self.effects); i++ {
			if self.effects[i].mode == DoublePass {
				resetBreakIndex = i
				break
			}
		}
	}

	// call LineBreak in reverse order until the end or until we hit the 
	// reset breakpoint for double pass effects
	for i := len(self.effects) - 1; i >= 0; i-- {
		advance := self.effects[i].CallLineBreak(renderer, target, self, position.X)
		if advance > 0 {
			iv.interruptKerning()
			position.X += renderer.withTextDirSign(advance)
		}

		// double pass reset case
		if i == resetBreakIndex {
			position, iv = self.resetForSecondPass(renderer, target, position, iv)
			return position, iv, true
		}
	}
	
	return position, iv, false
}

func (self *twineOperator) resetForSecondPass(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues) {
	// safety assertion (could be removed later, but it's cheap)
	if self.doublePassResetMode == dpResetNone || !self.onMeasuringPass || !self.performDrawPass {
		panic("misuse of twineOperator.resetForSecondPass")
	}

	self.onMeasuringPass = false

	position.X = self.doublePassResetX
	self.effects = self.effects[ : self.doublePassResetEffectIndex + 1]
	self.index = self.doublePassResetIndex
	iv.prevGlyphIndex = self.doublePassResetPrevGlyphIndex
	iv.lineBreakNth   = int(self.doublePassResetLineBreakNth)
	self.inGlyphMode  = self.doublePassResetInGlyphMode
	
	// call all active effects at the start of the reset mode
	switch self.doublePassResetMode {
	case dpResetLineStart:
		// multiple double pass effects can be accumulated from a previous line,
		// so we have to call line start from the first double pass effect
		// (not every effect, because single pass effects at the start of the
		// line would have already been drawn directly)
		doublePassEngaged := false
		for i := 0; i < len(self.effects); i++ {
			if doublePassEngaged || self.effects[i].mode == DoublePass {
				doublePassEngaged = true
				advance := self.effects[i].CallLineStart(renderer, target, self, position)
				if advance > 0 {
					iv.interruptKerning() // this might seem like it can be moved out here, but nope
					position.X += renderer.withTextDirSign(advance)
				}
			}
		}
	case dpResetPush:
		// in this case we can guarantee we have a single double pass effect,
		// and previous effects can't be anything else than single pass effects
		// that were already running in draw mode
		for i := 0; i < len(self.effects); i++ {
			if self.effects[i].mode == DoublePass {
				advance := self.effects[i].CallPush(renderer, target, self, position)
				if advance > 0 {
					iv.interruptKerning()
					position.X += advance
				}
				if i != len(self.effects) - 1 { panic("wat") }
				return position, iv
			}
		}
		panic("unreachable")
	default:
		panic("broken code")
	}

	self.doublePassResetMode = dpResetNone
	return position, iv
}

// This function checks whether we have to store any information when adding a 
// double pass effect or not, if we have to switch to measure mode or what.
func (self *twineOperator) notifyAddedDoublePassEffect(position fract.Point, iv drawInternalValues) {
	if self.doublePassResetMode == dpResetNone && self.performDrawPass {
		self.onMeasuringPass = true
		self.storeDoublePassResetData(position, iv, dpResetPush)
	}
}

func (self *twineOperator) storeDoublePassResetData(position fract.Point, iv drawInternalValues, resetMode dpResetMode) {
	self.doublePassResetMode = resetMode
	self.doublePassResetEffectIndex = uint16(len(self.effects))
	self.doublePassResetX = position.X	
	self.doublePassResetIndex = self.index
	self.doublePassResetLineBreakNth = int32(iv.lineBreakNth)
	self.doublePassResetPrevGlyphIndex = iv.prevGlyphIndex
	self.doublePassResetInGlyphMode = self.inGlyphMode
}
