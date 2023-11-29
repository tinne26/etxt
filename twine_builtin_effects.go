package etxt

import "image/color"

//import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// implements EffectPushColor
func twineEffectPushColor(renderer *Renderer, target Target, args TwineEffectArgs) fract.Unit {
	// usage asserts
	args.AssertPayloadLen(4)
	args.AssertMode(SinglePass)
	
	// bypass if measuring
	if args.Measuring() { return 0 }

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush:
		renderer.twineStoragePush(renderer.GetColor())
		r, g, b, a := args.Payload[0], args.Payload[1], args.Payload[2], args.Payload[3]
		renderer.SetColor(color.RGBA{r, g, b, a})
	case TwineTriggerPop:
		renderer.SetColor(renderer.twineStoragePop().(color.Color))
	case TwineTriggerLineBreak, TwineTriggerLineStart:
		// unused, not necessary
	default:
		panic("unexpected")
	}

	return 0
}

// implements EffectPushFont
func twineEffectPushFont(renderer *Renderer, target Target, args TwineEffectArgs) fract.Unit {
	// usage asserts
	args.AssertPayloadLen(1)
	args.AssertMode(SinglePass)

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush:
		renderer.twineStoragePush(renderer.state.fontIndex)
		renderer.Twine().SetFontIndex(FontIndex(args.Payload[0]))
	case TwineTriggerPop:
		index := renderer.twineStoragePop().(FontIndex)
		renderer.Twine().SetFontIndex(index)
	case TwineTriggerLineBreak, TwineTriggerLineStart:
		// unused, not necessary
	default:
		panic("unexpected")
	}

	return 0
}

// implements EffectShiftSize
func twineEffectShiftSize(renderer *Renderer, target Target, args TwineEffectArgs) fract.Unit {
	// usage asserts
	args.AssertPayloadLen(1)
	args.AssertMode(SinglePass)

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush:
		renderer.twineStoragePush(renderer.state.logicalSize)
		sizeShift := fract.FromInt(int(int8(args.Payload[0])))
		renderer.Fract().SetSize(renderer.state.logicalSize + sizeShift)
	case TwineTriggerPop:
		size := renderer.twineStoragePop().(fract.Unit)
		renderer.Fract().SetSize(size)
	case TwineTriggerLineBreak, TwineTriggerLineStart:
		// unused, not necessary
	default:
		panic("unexpected")
	}

	return 0
}
