//go:build !gtxt

package etxt

import "image/color"

//import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

func (self *Renderer) twineStoragePush(value any) {
	self.twineStorage = append(self.twineStorage, value)
}

func (self *Renderer) twineStoragePop() any {
	last := len(self.twineStorage) - 1
	value := self.twineStorage[last]
	self.twineStorage = self.twineStorage[ : last]
	return value
}

// implements EffectPushColor
func twineEffectPushColor(
	renderer *Renderer, payload []byte, flags TwineEffectFlags,
	target Target, _ fract.Point, _ fract.Rect,
) fract.Unit {
	// ignore on measuring
	if flags.IsMeasure() { return 0 }

	// some safety asserts
	if len(payload) != 4 { panic("unexpected") }
	if flags.IsPre() { panic("unexpected") }

	// push or pop
	switch flags.GetTrigger() {
	case TwineTriggerPush:
		renderer.twineStoragePush(renderer.GetColor())
		renderer.SetColor(color.RGBA{payload[0], payload[1], payload[2], payload[3]})
	case TwineTriggerPop:
		renderer.SetColor(renderer.twineStoragePop().(color.Color))
	default:
		panic("unexpected")
	}
	return 0
}

// implements EffectPushFont
func twineEffectPushFont(
	renderer *Renderer, payload []byte, flags TwineEffectFlags,
	target Target, _ fract.Point, _ fract.Rect,
) fract.Unit {
	// some safety asserts
	if len(payload) != 1 { panic("unexpected") }
	if flags.IsPre() { panic("unexpected") }

	// push or pop
	switch flags.GetTrigger() {
	case TwineTriggerPush:
		renderer.twineStoragePush(renderer.state.fontIndex)
		renderer.Complex().SetFontIndex(FontIndex(payload[0]))
	case TwineTriggerPop:
		index := renderer.twineStoragePop().(FontIndex)
		renderer.Complex().SetFontIndex(index)
	default:
		panic("unexpected")
	}
	return 0
}
