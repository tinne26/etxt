package etxt

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
	softPopped bool
}

func (self *effectOperationData) CallLineStart(renderer *Renderer, target Target, operator *twineOperator, newPosition fract.Point) fract.Unit {
	self.origin = newPosition

	flags := uint8(TwineTriggerLineStart)
	if operator.onMeasuringPass {
		self.forceLineBreakPostPad = false
		self.knownWidth = 0
		// NOTE: we don't consider min width spacing here because that's always used on
		//       push. The only thing is that push may use LineStart pre pad due to not
		//       fitting on the initial space.
	}

	var lineStartPad fract.Unit
	var postPad fract.Unit
	if self.spacing != nil {
		lineStartPad = self.spacing.getLineStartPad(renderer.state.scaledSize)
		self.origin.X += lineStartPad

		if !operator.onMeasuringPass {
			if self.forceLineBreakPostPad {
				postPad = self.spacing.getLineBreakPad(renderer.state.scaledSize)
			} else {
				postPad = self.spacing.getPostPad(renderer.state.scaledSize)
			}
		}
	}
	
	// invoke function and return advance
	self.commonCall(renderer, target, operator, self.origin.X, lineStartPad, postPad, flags)
	return lineStartPad
}

func (self *effectOperationData) CallPush(renderer *Renderer, target Target, operator *twineOperator, origin fract.Point) fract.Unit {
	self.origin = origin
	self.knownWidth = 0

	flags := uint8(TwineTriggerPush)
	if operator.onMeasuringPass {
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
		
		if operator.onMeasuringPass {
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

	// invoke function and return new x position
	self.commonCall(renderer, target, operator, self.origin.X, prePad, postPad, flags)
	return prePad
}

func (self *effectOperationData) CallPop(renderer *Renderer, target Target, operator *twineOperator, x fract.Unit) fract.Unit {
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
	self.commonCall(renderer, target, operator, x, prePad, postPad, flags)
	return postPad
}

func (self *effectOperationData) CallLineBreak(renderer *Renderer, target Target, operator *twineOperator, x fract.Unit) fract.Unit {
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
	self.commonCall(renderer, target, operator, x, prePad, postPad, flags)
	return postPad
}

func (self *effectOperationData) commonCall(renderer *Renderer, target Target, operator *twineOperator, x, prePad, postPad fract.Unit, flags uint8) {
	// obtain effect function
	var fn TwineEffectFunc
	if self.key > 192 { // built-in functions
		switch TwineEffectKey(self.key) {
		case EffectPushColor : fn = twineEffectPushColor
		case EffectPushFont  : fn = twineEffectPushFont
		case EffectShiftSize : fn = twineEffectShiftSize
		default:
			panic("private TwineEffectFunc #" + strconv.Itoa(int(self.key)) + " is not a defined built-in")
		}
	} else { // custom functions
		fn = renderer.twineEffectFuncs[self.key]
	}
	
	// misc. calculations
	if x - self.origin.X > self.knownWidth {
		self.knownWidth = x - self.origin.X
	}

	var payload []byte
	if self.payloadEndIndex > self.payloadStartIndex {
		payload = operator.twine.Buffer[self.payloadStartIndex : self.payloadEndIndex]
	}

	if self.mode == DoublePass {
		flags |= twineFlagDoublePass
	}
	if operator.onMeasuringPass {
		flags |= twineFlagMeasuring
	}

	// invoke effect function with the relevant arguments
	fn(renderer, target, TwineEffectArgs{
		Payload: payload,
		Origin: self.origin,
		LineAscent: operator.lineAscent,
		LineDescent: operator.lineDescent,
		KnownWidth: self.knownWidth,
		PrePad: prePad,
		KnownPostPad: postPad,
		flags: flags,
	})
}
