package etxt

import "fmt"
import "strconv"

import "github.com/tinne26/etxt/fract"

// The effectOperationData is a helper struct used for twine operations that
// allows us to store relevant data for calling the TwineEffectFuncs. To get
// started, see the twine.go file.
//
// Notice: any Call* operation that returns non-zero advance must interrupt
// kerning (drawInternalValues.interruptKerning()).
type effectOperationData struct {
	payloadStartIndex uint32 // inclusive
	payloadEndIndex uint32 // non-inclusive
	spacing *TwineEffectSpacing
	origin fract.Point
	knownWidth fract.Unit // doesn't include spacing pads
	forceLineBreakPostPad bool // if true, we use LineBreak post pad metrics
	forceLineStartPrePad bool // TODO: set manually on DrawWithWrap
	mode TwineEffectMode // ~bool
	key uint8

	// fields for twineOperatorEffectsList, should be manipulated 
	// only through that struct
	linkPrev uint16 // if prev == 65535, next indicates the next free index
	linkNext uint16
	status twineEffectListEntryStatus
}

func (self *effectOperationData) String() string {
	return fmt.Sprintf("effectOperationData{ payloadIndices: %d-%d, spacing: %t, key: %d, %s, %s, prev/next: %d/%d }",
		self.payloadStartIndex, self.payloadEndIndex, (self.spacing != nil), self.key, self.mode.string(), self.status, self.linkPrev, self.linkNext)
}

func (self *effectOperationData) CallLineStart(renderer *Renderer, target Target, measuring bool, twine *Twine, lineAscent, lineDescent fract.Unit, newPosition fract.Point) fract.Unit {
	//fmt.Printf("CallLineStart(%s) | %s\n", self.mode.string(), operator.passTypeStr())
	self.origin = newPosition

	flags := uint8(TwineTriggerLineStart)
	if measuring {
		self.forceLineBreakPostPad = false
		// NOTE: we don't consider min width spacing here because that's always used on
		//       push. The only thing is that push may use LineStart pre pad due to not
		//       fitting on the initial space.
	}

	var lineStartPad fract.Unit
	var postPad fract.Unit
	if self.spacing != nil {
		lineStartPad = self.spacing.getLineStartPad(renderer.state.scaledSize)
		self.origin.X += renderer.withTextDirSign(lineStartPad)

		if !measuring {
			if self.forceLineBreakPostPad {
				postPad = self.spacing.getLineBreakPad(renderer.state.scaledSize)
			} else {
				postPad = self.spacing.getPostPad(renderer.state.scaledSize)
			}
		}
	}

	// compensate origin with start pad
	self.origin.X += renderer.withTextDirSign(lineStartPad)
	
	// invoke function and return advance
	self.commonCall(renderer, target, measuring, twine, lineAscent, lineDescent, self.origin.X, lineStartPad, postPad, flags)
	return lineStartPad
}

func (self *effectOperationData) CallPush(renderer *Renderer, target Target, measuring bool, twine *Twine, lineAscent, lineDescent fract.Unit, origin fract.Point) fract.Unit {
	//fmt.Printf("CallPush(%s) | %s\n", self.mode.string(), operator.passTypeStr())
	self.origin = origin

	flags := uint8(TwineTriggerPush)
	if measuring {
		self.knownWidth = 0
		self.forceLineBreakPostPad = false
	}
	
	var prePad fract.Unit
	var postPad fract.Unit
	if self.spacing != nil {
		if self.forceLineStartPrePad {
			prePad = self.spacing.getLineStartPad(renderer.state.scaledSize)
		} else {
			prePad = self.spacing.getPrePad(renderer.state.scaledSize)
		}
		
		if measuring {
			minWidth := self.spacing.getMinWidth(renderer.state.scaledSize)
			if self.knownWidth < minWidth { self.knownWidth = minWidth } // not 100% sure
		} else {
			if self.forceLineBreakPostPad {
				postPad = self.spacing.getLineBreakPad(renderer.state.scaledSize)
			} else {
				postPad = self.spacing.getPostPad(renderer.state.scaledSize)
			}
		}
	}

	// compensate origin with pre pad
	self.origin.X += renderer.withTextDirSign(prePad)

	// invoke function and return new x position
	self.commonCall(renderer, target, measuring, twine, lineAscent, lineDescent, self.origin.X, prePad, postPad, flags)
	return prePad
}

func (self *effectOperationData) CallPop(renderer *Renderer, target Target, measuring bool, twine *Twine, lineAscent, lineDescent, x fract.Unit) fract.Unit {
	//fmt.Printf("CallPop(%s) | %s\n", self.mode.string(), operator.passTypeStr())
	if !measuring { self.knownWidth = 0 }

	var prePad fract.Unit
	var postPad fract.Unit
	if self.spacing != nil {
		postPad = self.spacing.getPostPad(renderer.state.scaledSize)
		if self.forceLineStartPrePad {
			prePad = self.spacing.getLineStartPad(renderer.state.scaledSize)
		} else {
			prePad = self.spacing.getPrePad(renderer.state.scaledSize)
		}
	}

	// invoke function and return new x position
	flags := uint8(TwineTriggerPop)
	self.commonCall(renderer, target, measuring, twine, lineAscent, lineDescent, x, prePad, postPad, flags)
	return postPad
}

func (self *effectOperationData) CallLineBreak(renderer *Renderer, target Target, measuring bool, twine *Twine, lineAscent, lineDescent, x fract.Unit) fract.Unit {
	//fmt.Printf("CallLineBreak(%s) | %s\n", self.mode.string(), operator.passTypeStr())
	if !measuring { self.knownWidth = 0 }
	self.forceLineBreakPostPad = true

	var prePad fract.Unit
	var postPad fract.Unit
	if self.spacing != nil {
		postPad = self.spacing.getLineBreakPad(renderer.state.scaledSize)
		if self.forceLineStartPrePad {
			prePad = self.spacing.getLineStartPad(renderer.state.scaledSize)
		} else {
			prePad = self.spacing.getPrePad(renderer.state.scaledSize)
		}
	}


	// invoke function and return new x position
	flags := uint8(TwineTriggerLineBreak)
	self.commonCall(renderer, target, measuring, twine, lineAscent, lineDescent, x, prePad, postPad, flags)
	return postPad
}

func (self *effectOperationData) commonCall(renderer *Renderer, target Target, measuring bool, twine *Twine, lineAscent, lineDescent, x, prePad, postPad fract.Unit, flags uint8) {
	// obtain effect function
	var fn TwineEffectFunc
	if self.key > 192 { // built-in functions
		switch TwineEffectKey(self.key) {
		case EffectPushColor  : fn = twineEffectPushColor
		case EffectPushFont   : fn = twineEffectPushFont
		case EffectShiftSize  : fn = twineEffectShiftSize
		case EffectSetSize    : fn = twineEffectSetSize
		case EffectOblique    : fn = twineEffectOblique
		case EffectFauxBold   : fn = twineEffectFauxBold
		case EffectHighlightA : fn = twineEffectHighlightA
		case EffectHighlightB : fn = twineEffectHighlightB
		case EffectCrossOut   : fn = twineEffectCrossOut
		default:
			panic("private TwineEffectFunc #" + strconv.Itoa(int(self.key)) + " is not a defined built-in")
		}
	} else { // custom functions
		fn = renderer.twineEffectFuncs[self.key]
	}
	
	// misc. calculations
	if renderer.state.textDirection == LeftToRight { // LTR
		if x - self.origin.X > self.knownWidth {
			self.knownWidth = x - self.origin.X
		}
	} else { // RTL
		flags |= twineFlagRightToLeft
		if self.origin.X - x > self.knownWidth {
			self.knownWidth = self.origin.X - x
		}
	}

	var payload []byte
	if self.payloadEndIndex > self.payloadStartIndex {
		payload = twine.Buffer[self.payloadStartIndex : self.payloadEndIndex]
	}

	if measuring { flags |= twineFlagMeasuring }
	if self.mode == DoublePass {
		flags |= twineFlagDoublePass
	}

	// invoke effect function with the relevant arguments
	fn(renderer, target, TwineEffectArgs{
		Payload: payload,
		Origin: self.origin,
		LineAscent: lineAscent,
		LineDescent: lineDescent,
		KnownWidth: self.knownWidth,
		PrePad: prePad,
		KnownPostPad: postPad,
		flags: flags,
	})
}
