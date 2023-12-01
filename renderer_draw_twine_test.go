//go:build gtxt

package etxt

import "testing"
import "strconv"
import "image"

import "github.com/tinne26/etxt/fract"

func consistentArgs(a, b TwineEffectArgs) bool {
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
	return "TwineEffectArgs{ Payload [" + strconv.Itoa(len(tea.Payload)) + "] ; Rect " + tea.Rect().String() + 
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
	
	// register effect func
	renderer.Twine().RegisterEffectFunc(0, tester.EffectFunc)

	// (no effects)
	tester.Init([]TwineEffectArgs{})	
	twine.Reset()
	twine.Add("one two three")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Effect func test #0 failed: %s", tester.ErrMsg())
	}

	// check simplest effect function case
	// (SinglePass, push/pop, no multiline, no advances, no payload)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush),
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop),
		},
	})
	twine.Reset()
	twine.Add("one ").PushEffect(0, SinglePass).Add("two ").Pop().Add("three")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Effect func test #1 failed: %s", tester.ErrMsg())
	}

	// (SinglePass, push/pop, no multiline, no advances, payload)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: []byte{1, 2, 3},
			flags: uint8(TwineTriggerPush),
		},
		TwineEffectArgs{
			Payload: []byte{1, 2, 3},
			flags: uint8(TwineTriggerPop),
		},
	})
	twine.Reset()
	twine.Add("one ").PushEffect(0, SinglePass, 1, 2, 3).Add("two ").Pop().Add("three")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Effect func test #2 failed: %s", tester.ErrMsg())
	}

	// (SinglePass, push/pop, multiline, no advances, payload)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: []byte{3, 6},
			flags: uint8(TwineTriggerPush),
		},
		TwineEffectArgs{
			Payload: []byte{3, 6},
			flags: uint8(TwineTriggerLineBreak),
		},
		TwineEffectArgs{
			Payload: []byte{3, 6},
			flags: uint8(TwineTriggerLineStart),
		},
		TwineEffectArgs{
			Payload: []byte{3, 6},
			flags: uint8(TwineTriggerPop),
		},
	})
	twine.Reset()
	twine.Add("please ").PushEffect(0, SinglePass, 3, 6).Add("don't\nmove").Pop().Add(" like that")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Effect func test #3 failed: %s", tester.ErrMsg())
	}

	// (DoublePass, push/pop, no multiline, no advances, no payload)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass,
		},	
	})
	t.Log("\n--- double pass test ---\n")
	twine.Reset()
	twine.Add("double ").PushEffect(0, DoublePass).Add("pass").Pop().Add(" mode")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Effect func test #4 failed: %s", tester.ErrMsg())
	}

	// TODO: test dynamic payload modification

	// TODO: loop for testing under every possible align and text direction
	//       (wait, but behavior will actually change with this. I can change
	//       vertical aligns, but horizontal aligns + text directions will
   //       necessarily cause some amount of chaos)
	// TODO: line size changes have to be tested.

	// TODO: test LineStart trigger on double pass rewind? is that even a thing?

	// TODO: test reset of single pass effect which begins in draw mode but
	//       then we get to a double pass effect that changes mode in between
	//       and has a line break. this would force the measure close for 
	//       the doublepass effect, start the second draw pass for the double
	//       pass, and only then close them both from their draw mode. so,
	//       the single pass effect is surprisingly not notified about the
	//       measuring pass in draw mode.

	// TODO: test for the case where we have a double pass effect encompassing multiple
	//       line breaks consecutively, to make sure we don't accidentally break
	//       lineBreakNth sequences when not really necessary (and that we do when
	//       necessary)

	// TODO: use coverage to add remaining tests
}
