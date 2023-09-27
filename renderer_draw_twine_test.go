//go:build gtxt

package etxt

import "testing"
import "strconv"
import "image"

import "github.com/tinne26/etxt/fract"

func consistentArgs(a, b TwineEffectArgs) bool {
	// if a.Rect != b.Rect { return false }
	// if a.Origin != b.Origin { return false }
	if !eqByteSlices(a.Payload, b.Payload) { return false }
	if a.flags != b.flags { return false }
	return true
}

func eqByteSlices(a, b []byte) bool {
	if len(a) != len(b) { return false }
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] { return false }
	}
	return true
}

func twineEffectArgsStr(tea TwineEffectArgs) string {
	return "TwineEffectArgs{ Payload [" + strconv.Itoa(len(tea.Payload)) + "] ; Rect " + tea.Rect.String() + 
		" ; Origin " + tea.Origin.String() + " ; flags " + strconv.FormatUint(uint64(tea.flags), 2) + " }"
}

type twineEffectTester struct {
	expected []TwineEffectArgs
	index int
	errMsg string
}

func (self *twineEffectTester) Init(expected []TwineEffectArgs) {
	self.expected = expected
	self.index = 0
	self.errMsg = ""
}

func (self *twineEffectTester) EffectFunc(renderer *Renderer, target Target, args TwineEffectArgs) fract.Unit {
	if self.errMsg != "" { return 0 }

	if self.index >= len(self.expected) {
		self.errMsg = "unexpected call to effect func at invocation#" + strconv.Itoa(self.index) +
			" with " + twineEffectArgsStr(args) + " (expected less invocations)"
		return 0
	}
	
	if !consistentArgs(args, self.expected[self.index]) {
		self.errMsg = "inconsistent arguments to effect func at invocation#" + strconv.Itoa(self.index) +
			"; expected " + twineEffectArgsStr(self.expected[self.index]) + ", got " + twineEffectArgsStr(args)
		return 0
	}

	self.index += 1
	return 0
}

func (self *twineEffectTester) EndSequence() {
	if self.index < len(self.expected) {
		self.errMsg = "effect func expected " + strconv.Itoa(len(self.expected)) + " invocations, but got only " +
			strconv.Itoa(self.index)
	}
}

func (self *twineEffectTester) HasError() bool { return self.errMsg != "" }
func (self *twineEffectTester) ErrMsg() string { return self.errMsg }

func TestDrawTwineEffects(t *testing.T) {
	if testFontA == nil { t.SkipNow() }

	renderer := NewRenderer()
	renderer.SetFont(testFontA)
	renderer.Utils().SetCache8MiB()

	// create tester
	var tester twineEffectTester
	var twine Twine
	target := image.NewRGBA(image.Rect(0, 0, 640, 480))

	// check simplest effect function case
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDraw,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDraw | twineFlagRectOk,
		},
	})
	renderer.Complex().RegisterEffectFunc(0, tester.EffectFunc)
	twine.Add("one ").PushEffect(0).Add("two ").Pop().Add("three")
	renderer.Complex().Draw(target, twine, 32, 32)

	if tester.HasError() {
		t.Fatalf("Effect func test #1 failed: %s", tester.ErrMsg())
	}

	// TODO: test dynamic payload modification
}
