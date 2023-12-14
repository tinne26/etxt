//go:build !gtxt

package etxt

import "strconv"
import "image/color"

// implements EffectHighlightA (crisp filled rect slightly shifted down)
func twineEffectHighlightA(renderer *Renderer, target Target, args TwineEffectArgs) {
	// usage asserts
	args.AssertMode(DoublePass)
	var clr color.RGBA
	switch len(args.Payload) {
	case 3: // RGB
		clr.R, clr.G, clr.B, clr.A = args.Payload[0], args.Payload[1], args.Payload[2], 255
	case 4: // RGBA
		clr.R, clr.G, clr.B, clr.A = args.Payload[0], args.Payload[1], args.Payload[2], args.Payload[3]
	default:
		payloadLenStr := strconv.Itoa(len(args.Payload))
		panic(
			"EffectHighlightA expects 3 or 4 bytes as the payload (RGB or RGBA bytes), " +
			"but got " + payloadLenStr + " bytes instead.",
		)
	}
	
	// bypass if measuring
	if args.Measuring() { return }

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush, TwineTriggerLineStart:
		rect := args.RectWithPads()
		xheightApprox := (args.LineAscent).Mul(32)
		rect.Min.Y = args.Origin.Y - (xheightApprox >> 1) + (xheightApprox >> 4)
		rect.Max.Y = args.Origin.Y + args.LineDescent + (xheightApprox >> 3)
		fillOver(rect.Clip(target), clr)
	case TwineTriggerPop, TwineTriggerLineBreak:
		// nothing to do here
	default:
		panic("unexpected")
	}
}

func twineEffectCrossOut(renderer *Renderer, target Target, args TwineEffectArgs) {
	// usage asserts
	args.AssertMode(SinglePass)
	var thicknessPercent float32
	switch len(args.Payload) {
	case 0: thicknessPercent = 0.5
	case 1: thicknessPercent = (float32(args.Payload[0]) + 1.0)/256.0
	default:
		payloadLenStr := strconv.Itoa(len(args.Payload))
		panic(
			"EffectCrossOut expects one byte in the payload or no bytes " + 
			"at all (defaults to 128), but got " + payloadLenStr + " bytes " + 
			"instead.",
		)
	}

	// bypass if measuring
	if args.Measuring() { return }

	// handle each trigger situation
	switch args.GetTrigger() {
	case TwineTriggerPush, TwineTriggerLineStart:
		// nothing to do here
	case TwineTriggerPop, TwineTriggerLineBreak:
		crossOutLineCenter := (args.Origin.Y - args.LineAscent.Mul(16)).ToFloat32()
		halfLineWidth := renderer.state.scaledSize.ToFloat32()*thicknessPercent*0.12
		if halfLineWidth < 0.3 { halfLineWidth = 0.3 } // arbitrary safety value

		var minX, maxX float32
		if args.IsLeftToRight() {
			minX = (args.Origin.X - args.PrePad).ToFloat32()
			maxX = (args.Origin.X + args.KnownWidth + args.KnownPostPad).ToFloat32()
		} else {
			minX = (args.Origin.X - args.KnownWidth - args.KnownPostPad).ToFloat32()
			maxX = (args.Origin.X + args.PrePad).ToFloat32()
		}
		minY := crossOutLineCenter - halfLineWidth
		maxY := crossOutLineCenter + halfLineWidth
		drawSmoothRect(target, minX, minY, maxX, maxY, renderer.GetColor())
	default:
		panic("unexpected")
	}
}
