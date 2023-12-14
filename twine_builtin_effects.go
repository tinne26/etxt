package etxt

import "strconv"
import "image/color"

//import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"

// implements EffectPushColor
func twineEffectPushColor(renderer *Renderer, target Target, args TwineEffectArgs) {
	// usage asserts
	args.AssertPayloadLen(4)
	args.AssertMode(SinglePass)
	
	// bypass if measuring
	if args.Measuring() { return }

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
}

// implements EffectPushFont
func twineEffectPushFont(renderer *Renderer, target Target, args TwineEffectArgs) {
	// usage asserts
	args.AssertPayloadLen(1)
	args.AssertMode(SinglePass)

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush, TwineTriggerLineStart:
		renderer.twineStoragePush(renderer.state.fontIndex)
		renderer.Twine().SetFontIndex(FontIndex(args.Payload[0]))
	case TwineTriggerPop, TwineTriggerLineBreak:
		index := renderer.twineStoragePop().(FontIndex)
		renderer.Twine().SetFontIndex(index)
	default:
		panic("unexpected")
	}
}

// implements EffectShiftSize
func twineEffectShiftSize(renderer *Renderer, target Target, args TwineEffectArgs) {
	// usage asserts
	args.AssertPayloadLen(1)
	args.AssertMode(SinglePass)

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush, TwineTriggerLineStart:
		renderer.twineStoragePush(renderer.state.logicalSize)
		sizeShift := fract.FromInt(int(int8(args.Payload[0])))
		renderer.Fract().SetSize(renderer.state.logicalSize + sizeShift)
	case TwineTriggerPop, TwineTriggerLineBreak:
		size := renderer.twineStoragePop().(fract.Unit)
		renderer.Fract().SetSize(size)
	default:
		panic("unexpected")
	}
}

// implements EffectSetSize
func twineEffectSetSize(renderer *Renderer, target Target, args TwineEffectArgs) {
	// usage asserts
	args.AssertPayloadLen(1)
	args.AssertMode(SinglePass)

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush, TwineTriggerLineStart:
		renderer.twineStoragePush(renderer.state.logicalSize)
		newSize := fract.FromInt(int(args.Payload[0]))
		renderer.Fract().SetSize(newSize)
	case TwineTriggerPop, TwineTriggerLineBreak:
		size := renderer.twineStoragePop().(fract.Unit)
		renderer.Fract().SetSize(size)
	default:
		panic("unexpected")
	}
}

// implements EffectOblique
func twineEffectOblique(renderer *Renderer, target Target, args TwineEffectArgs) {
	// usage asserts
	args.AssertMode(SinglePass)
	fauxRast, isFauxRast := renderer.Glyph().GetRasterizer().(*mask.FauxRasterizer)
	if !isFauxRast {
		panic("EffectOblique requires using the mask.FauxRasterizer")
	}
	var angleReference byte
	switch len(args.Payload) {
	case 0: angleReference = 196
	case 1: angleReference = args.Payload[0]
	default:
		payloadLenStr := strconv.Itoa(len(args.Payload))
		panic(
			"EffectOblique expects one byte (below 128 for left slant, above " + 
			"128 for right slant) in the payload or no bytes at all (defaults " + 
			"to 196), but got " + payloadLenStr + " bytes instead.",
		)
	}

	// bypass if measuring
	if args.Measuring() { return }

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush:
		renderer.twineStoragePush(fauxRast.GetSkewFactor())
		if angleReference == 128 { // (no slant)
			fauxRast.SetSkewFactor(0)
		} else if angleReference > 128 { // (right slant)
			fauxRast.SetSkewFactor(0.3*(float32(angleReference - 128)/128.0))
		} else { // angleReference < 128 // (left slant)
			fauxRast.SetSkewFactor(-0.3*float32(128 - angleReference)/128.0)
		}
	case TwineTriggerPop:
		fauxRast.SetSkewFactor(renderer.twineStoragePop().(float32))
	case TwineTriggerLineBreak, TwineTriggerLineStart:
		// unused, not necessary
	default:
		panic("unexpected")
	}
}

// implements EffectFauxBold
func twineEffectFauxBold(renderer *Renderer, target Target, args TwineEffectArgs) {
	// usage asserts
	args.AssertMode(SinglePass)
	fauxRast, isFauxRast := renderer.Glyph().GetRasterizer().(*mask.FauxRasterizer)
	if !isFauxRast {
		panic("EffectFauxBold requires using the mask.FauxRasterizer")
	}
	var thicknessReference byte
	switch len(args.Payload) {
	case 0: thicknessReference = 128
	case 1: thicknessReference = args.Payload[0]
	default:
		payloadLenStr := strconv.Itoa(len(args.Payload))
		panic(
			"EffectFauxBold expects one byte in the payload or no bytes " + 
			"at all (defaults to 128), but got " + payloadLenStr + " bytes " + 
			"instead.",
		)
	}
	
	// bypass if measuring
	if args.Measuring() { return }

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush:
		renderer.twineStoragePush(fauxRast.GetExtraWidth())
		const thicknessBasePercent = 0.14
		factor := (float32(thicknessReference) + 1.0)/256.0
		factor *= thicknessBasePercent
		extraWidth := renderer.state.scaledSize.ToFloat32()*factor
		fauxRast.SetExtraWidth(extraWidth)
	case TwineTriggerPop:
		fauxRast.SetExtraWidth(renderer.twineStoragePop().(float32))
	case TwineTriggerLineBreak, TwineTriggerLineStart:
		// unused, not necessary
	default:
		panic("unexpected")
	}
}
