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
	twine Twine // the twine we are operating on
	index int // current index for twine.Buffer
	inGlyphMode bool // twines can mix glyphs and utf8, so we need to keep track of this
	performDrawPass bool // whether the operator must draw or only measure
	onMeasuringPass bool // whether we are on a measuring or drawing pass
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

	// - double pass reset values -
	doublePassResetMode dpResetMode
	doublePassResetInGlyphMode bool
	doublePassResetX fract.Unit
	doublePassResetLineBreakNth int32
	doublePassResetPrevGlyphIndex sfnt.GlyphIndex // uint16, 2 bytes
	doublePassResetIndex int // index to jump back after first measuring pass.
	                         // the index is placed after the effect's CC and payload
}

// Must be called before starting to use it. Resets anything left from previous uses, if any.
func (self *twineOperator) Initialize(renderer *Renderer, twine Twine, mustDraw bool, yCutoff fract.Unit) {
	self.twine = twine
	self.performDrawPass = mustDraw
	self.onMeasuringPass = !mustDraw
	self.defaultNewLineX = 0
	self.shiftNewLineX = 0
	self.yCutoff  = yCutoff
	self.effects.Initialize()
	
	// core resets
	self.index = 0
	self.inGlyphMode = false // twines start in utf8 mode
	self.doublePassResetMode = dpResetNone
	
	// get initial line metrics
	self.registerNextLineMetrics(renderer)
	self.newLineMetricsRefresh()
}

func (self *twineOperator) SetDefaultNewLineX(x fract.Unit) {
	self.defaultNewLineX = x
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

func (self *twineOperator) Operate(renderer *Renderer, target Target, position fract.Point, glyphIndex sfnt.GlyphIndex, iv drawInternalValues) (fract.Point, drawInternalValues) {
	if renderer.state.textDirection == LeftToRight {
		if self.onMeasuringPass {
			position.X, iv = renderer.advanceGlyphLTR(position.X, glyphIndex, iv)
			return position, iv
		} else { // drawing
			return renderer.drawGlyphLTR(target, position, glyphIndex, iv)
		}
	} else { // assume right to left
		if self.onMeasuringPass {
			position.X, iv = renderer.advanceGlyphRTL(position.X, glyphIndex, iv)
			return position, iv
		} else { // drawing
			return renderer.drawGlyphRTL(target, position, glyphIndex, iv)
		}
	}
}

func (self *twineOperator) LineBreak(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues, bool) {
	// notify line end, which could trigger a double pass reset
	//fmt.Printf("trace: LineBreak() / %s\n", self.effects.debugStr())
	var dpReset bool
	position, iv, dpReset = self.notifyLineBreak(renderer, target, position, iv) // *
	if dpReset { return position, iv, true }
	// * notifyLineBreaks makes the CallLineBreak invokations, but it may
	//   also trigger the double reset pass, in which case it will also
	//   call the CallLineStart or CallPush methods

	// start setting everything up for next line
	iv.increaseLineBreakNth()
	
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

	// check y cutoff to terminate early and pop every remaining effect in that case
	if position.Y > self.yCutoff {
		//fmt.Print("trace: LineBreak() cutoff\n")
		position, iv, _ = self.PopAll(renderer, target, position, iv)
		return position, iv, false
	}

	// update/invoke effects for the new line
	if !self.performDrawPass || self.effects.ActiveDoublePassEffectsCount() == 0 {
		//fmt.Print("trace: LineBreak() / easy CallLineStart\n")
		// easy case: we either don't have to draw (so it's all just measuring,
		// though effects changing font or font size can still affect metrics),
		// or we have to draw but we don't have active double pass effects so
		// we can just call line start on everyone anyway.
		self.effects.AssertAllEffectsActive()
		self.effects.Each(func(effect *effectOperationData) {
			effect.origin = position
			advance := effect.CallLineStart(renderer, target, self.onMeasuringPass, &self.twine, self.lineAscent, self.lineDescent, position)
			if advance != 0 {
				iv.interruptKerning()
				position.X += renderer.withTextDirSign(advance)
			}
		})
	} else {
		//fmt.Print("trace: LineBreak() / hard pass with storeDoublePassReset\n")
		// super annoying tricky case! if we still have double pass effects accumulated,
		// we will need to memorize a new reset point and go into measure mode again...
		// but only once we hit the first double pass effect. single pass effects wrapping
		// from further back can already be "drawn" when hit.
		// 
		// the loop is similar to the previous branch, but with some extra logic (just
		// read the comments)
		if self.onMeasuringPass { panic("assertion") }
		self.effects.Each(func(effect *effectOperationData) {
			// if we find a double pass effect while still drawing,
			// it's time to switch to measure mode and set our reset point
			if !self.onMeasuringPass && effect.mode == DoublePass {
				self.onMeasuringPass = true
				self.storeDoublePassResetData(position, iv, dpResetLineStart)
			}

			// regular loop logic (update effect positions, call them with TwineTriggerStartLine)
			effect.origin = position
			advance := effect.CallLineStart(renderer, target, self.onMeasuringPass, &self.twine, self.lineAscent, self.lineDescent, position)
			if advance != 0 {
				iv.interruptKerning()
				position.X += renderer.withTextDirSign(advance)
			}
		})
		if !self.onMeasuringPass { panic("assertion") }
	}
	
	return position, iv, false
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
		self.index += 1
	case twineCcPushLineRestartMarker:
		self.shiftNewLineX = position.X - self.defaultNewLineX
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
		return self.ProcessCC(renderer, target, position, iv)
	case twineCcPop:
		position, iv, dpReset = self.pop(renderer, target, position, iv)
		if !dpReset { self.index += 1 } // otherwise the index is set on pop >> restorePreState
	case twineCcPopAll:
		position, iv, dpReset = self.PopAll(renderer, target, position, iv)
		if !dpReset { self.index += 1 }
	case twineCcPushSinglePassEffect:
		self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, SinglePass)
		asc, desc := self.lineAscent, self.lineDescent
		advance := self.effects.Head().CallPush(renderer, target, self.onMeasuringPass, &self.twine, asc, desc, position)
		if advance != 0 {
			iv.interruptKerning()
			position.X += renderer.withTextDirSign(advance)
		}
	case twineCcPushDoublePassEffect:
		self.index = self.appendNewEffectOpDataWithKeyAt(self.index + 1, DoublePass)
		
		// store double pass reset data if pushing the first 
		// double pass effect while in draw mode.
		if !self.onMeasuringPass && self.effects.ActiveDoublePassEffectsCount() == 1 {
			if self.doublePassResetMode != dpResetNone { panic("bad assumption") }
			self.onMeasuringPass = true
			self.storeDoublePassResetData(position, iv, dpResetPush)
		}

		// call push
		asc, desc := self.lineAscent, self.lineDescent
		advance := self.effects.Head().CallPush(renderer, target, self.onMeasuringPass, &self.twine, asc, desc, position)
		if advance != 0 {
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
	//fmt.Printf("trace: PopAll() / %s\n", self.effects.debugStr())
	var dpReset bool
	for self.effects.ActiveCount() > 0 {
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
	// if an effect is known to be fully unnecessary for the rest of the process
	// (which is only if !performDrawPass or finally in double pass draw) we can
	// effect.HardPop() it, otherwise we will have to use it again on the doublePass
	// so we only effect.SoftPop() it.
	if self.doublePassResetMode != dpResetNone && self.effects.OnLastDoublePassEffect() {
		if !self.onMeasuringPass { panic("expected to be on measuring pass, broken code") }
		//fmt.Printf("trace: soft pop() and reset / %s\n", self.effects.debugStr())
		// we reset to begin the second pass when we have a double pass reset mode 
		// configured, we are measuring and the pop the last double pass effect.
		effect := self.effects.SoftPop()
		_ = effect.CallPop(renderer, target, self.onMeasuringPass, &self.twine, self.lineAscent, self.lineDescent, position.X)
		position, iv = self.resetForSecondPass(renderer, target, position, iv)
		return position, iv, true
	} else if self.doublePassResetMode == dpResetNone || !self.onMeasuringPass {
		// we hard pop when we are not in reset mode (e.g. measuring only) or 
		// when we are drawing effect is no longer necessary
		//fmt.Printf("trace: hard pop() / %s\n", self.effects.debugStr())
		asc, desc := self.lineAscent, self.lineDescent
		advance := self.effects.Head().CallPop(renderer, target, self.onMeasuringPass, &self.twine, asc, desc, position.X)
		if advance != 0 {
			iv.interruptKerning()
			position.X += renderer.withTextDirSign(advance)
		}
		self.effects.HardPop()
	} else {
		// effect will be needed later
		//fmt.Printf("trace: soft pop() / %s\n", self.effects.debugStr())
		effect := self.effects.SoftPop()
		advance := effect.CallPop(renderer, target, self.onMeasuringPass, &self.twine, self.lineAscent, self.lineDescent, position.X)
		if advance != 0 {
			iv.interruptKerning()
			position.X += renderer.withTextDirSign(advance)
		}
	}

	return position, iv, false
}

// Appends a new effect op data value and returns the new index (first position after the payload).
// One must still invoke CallPush or CallLineStart on the effect afterwards.
func (self *twineOperator) appendNewEffectOpDataWithKeyAt(index int, effectMode TwineEffectMode) int {
	//fmt.Printf("trace: appendNewEffectOpData...(%s) / %s\n", effectMode.string(), self.effects.debugStr())
	// note: the data structure is [twineCcBegin, effectModeCc, key, payloadLen, payload...]
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
		opData.spacing = self.spacingPendingAdd
		self.spacingPendingAdd = nil

		// push op data
		self.effects.Push(opData)
	}

	// clear spacePendingAdd and return new twine buffer index
	self.spacingPendingAdd = nil
	return index + int(payloadLen) + 2
}

func (self *twineOperator) notifyLineBreak(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues, bool) {
	// we have to call LineBreak on most effects, but there's the possibility of a double
	// pass reset midway, before all effects have line end called on them. so, we want to
	// split the code in the simple case (no reset or already on second pass draw), and
	// the hard case where we only deal with a certain number of effects
	if self.doublePassResetMode == dpResetNone || !self.onMeasuringPass {
		// simple case, notify everyone of line break
		// (we don't pop any effect, as we need them on the next line)
		self.effects.EachReverse(func(effect *effectOperationData) {
			//fmt.Print("trace: notifyLineBreak() / CallLineBreak()\n")
			advance := effect.CallLineBreak(renderer, target, self.onMeasuringPass, &self.twine, self.lineAscent, self.lineDescent, position.X)
			if advance != 0 {
				iv.interruptKerning()
				position.X += renderer.withTextDirSign(advance)
			}
		})
		return position, iv, false
	} else {
		
		// hard case, notify everyone up to the furthest away double pass, then reset
		dpCount := self.effects.ActiveDoublePassEffectsCount()
		if dpCount == 0 { panic("broken code") } // TODO: delete later
		
		// soft pop everyone up to last double pass effect
		for dpCount > 0 {
			//fmt.Print("trace: notifyLineBreak() / SoftPop + CallLineBreak\n")
			effect := self.effects.SoftPop()
			advance := effect.CallLineBreak(renderer, target, self.onMeasuringPass, &self.twine, self.lineAscent, self.lineDescent, position.X)
			if advance != 0 {
				iv.interruptKerning()
				position.X += renderer.withTextDirSign(advance)
			}
			if effect.mode == DoublePass {
				dpCount -= 1
			}
		}

		position, iv = self.resetForSecondPass(renderer, target, position, iv)
		return position, iv, true
	}
}

// This can be invoked from a pop or a line break, if we have a relevant double 
// pass effect drawing pass pending. This function resets some operating values
// (twine index, position, glyph mode, etc) and then calls LineStart or Push
// for the relevant double pass effects.
func (self *twineOperator) resetForSecondPass(renderer *Renderer, target Target, position fract.Point, iv drawInternalValues) (fract.Point, drawInternalValues) {
	//fmt.Printf("trace: resetForSecondPass() / %s\n", self.effects.debugStr())
	// safety assertion (could be removed later, but it's cheap)
	if self.doublePassResetMode == dpResetNone || !self.onMeasuringPass || !self.performDrawPass {
		panic("misuse of twineOperator.resetForSecondPass")
	}

	self.onMeasuringPass = false
	position.X = self.doublePassResetX
	self.index = self.doublePassResetIndex
	iv.prevGlyphIndex = self.doublePassResetPrevGlyphIndex
	iv.lineBreakNth   = int(self.doublePassResetLineBreakNth)
	self.inGlyphMode  = self.doublePassResetInGlyphMode
	
	// recover last soft popped double pass effect
	effect := self.effects.TryRecallNext()
	if effect == nil { panic("broken code") } // TODO: delete later

	// call all active effects at the start of the reset mode
	switch self.doublePassResetMode {
	case dpResetLineStart:
		// recall any remaining effects that were wrapped in a multi-line
		for self.effects.TryRecallNext() != nil {}

		// multiple double pass effects can be accumulated from a previous line,
		// so we have to call line start from the oldest double pass effect still
		// active to the newest effect wrapped in the reset zone
		engaged := false
		self.effects.Each(func(effect *effectOperationData) {
			if engaged || effect.mode == DoublePass {
				//fmt.Print("trace: resetForSecondPass() / CallLineStart()\n")
				engaged = true
				advance := effect.CallLineStart(renderer, target, self.onMeasuringPass, &self.twine, self.lineAscent, self.lineDescent, position)
				if advance != 0 {
					iv.interruptKerning() // this might seem like it can be moved out here, but nope
					position.X += renderer.withTextDirSign(advance)
				}
			}
		})
	case dpResetPush:
		// in this case we can guarantee we have a single double pass effect,
		// and previous effects can't be anything else than single pass effects
		// that were already invoked in draw mode
		//fmt.Print("trace: resetForSecondPass() / Push()\n")
		advance := effect.CallPush(renderer, target, self.onMeasuringPass, &self.twine, self.lineAscent, self.lineDescent, position)
		if advance != 0 {
			iv.interruptKerning()
			position.X += advance
		}
	default:
		panic("broken code")
	}
	
	self.doublePassResetMode = dpResetNone
	return position, iv
}

func (self *twineOperator) storeDoublePassResetData(position fract.Point, iv drawInternalValues, resetMode dpResetMode) {
	//fmt.Printf("trace: storing double pass reset data / %s\n", self.effects.debugStr())
	self.doublePassResetMode = resetMode
	self.doublePassResetX = position.X	
	self.doublePassResetIndex = self.index
	self.doublePassResetLineBreakNth = int32(iv.lineBreakNth)
	self.doublePassResetPrevGlyphIndex = iv.prevGlyphIndex
	self.doublePassResetInGlyphMode = self.inGlyphMode
}

// ---- debug ----
func (self *twineOperator) passTypeStr() string {
	if self.onMeasuringPass { return "measuring" }
	return "drawing"
}
