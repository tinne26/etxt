//go:build gtxt

package etxt

import "testing"
import "strconv"
import "image"

// TODO:
// - test line size changes
// - test dynamic payload modification
// - test RTL
// - test double pass effects encompassing multiple consecutive
//   line breaks (lineBreakNth sequence breaking test)
// - comparative draws with simple strings against default algorithm

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

func (self *twineEffectTester) ResetIndexAndErr() {
	self.index = 0
	self.errMsg = ""
}

func (self *twineEffectTester) EffectFunc(renderer *Renderer, target Target, args TwineEffectArgs) {
	if self.errMsg != "" { return }

	if self.index >= len(self.expected) {
		self.errMsg = "unexpected call to effect func at invocation#" + strconv.Itoa(self.index) +
			" with " + twineEffectArgsStr(args) + " (expected less invocations)"
		return
	}
	
	if !consistentArgs(args, self.expected[self.index]) {
		self.errMsg = "inconsistent arguments to effect func at invocation#" + strconv.Itoa(self.index) +
			"; expected " + twineEffectArgsStr(self.expected[self.index]) + ", got " + twineEffectArgsStr(args)
		return
	}

	self.index += 1
}

func (self *twineEffectTester) EndSequence() {
	if self.HasError() { return } // keep previous error
	if self.index < len(self.expected) {
		self.errMsg = "effect func expected " + strconv.Itoa(len(self.expected)) + " invocations, but got only " +
			strconv.Itoa(self.index)
	}
}

func (self *twineEffectTester) HasError() bool { return self.errMsg != "" }
func (self *twineEffectTester) ErrMsg() string { return self.errMsg }

func TestDrawBasicTwineEffects(t *testing.T) {
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
	twine.Reset()
	twine.Add("double ").PushEffect(0, DoublePass).Add("pass").Pop().Add(" mode")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Effect func test #4 failed: %s", tester.ErrMsg())
	}

	// (DoublePass, push/pop, multiline, no advances, no payload)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass,
		},	
	})
	twine.Reset()
	twine.Add("double ").PushEffect(0, DoublePass).Add("pass\neffect").Pop().Add(" mode")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Effect func test #5 failed: %s", tester.ErrMsg())
	}

	// (DoublePass, push/pop, multiline, no advances, payload)
	tester.Init([]TwineEffectArgs{
		// first line: push, line break, again with second pass
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass,
		},

		// second line: line start, line end, again with second pass
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass,
		},

		// third line: line start, pop, again with second pass
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: []byte{22},
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass,
		},	
	})
	twine.Reset()
	twine.Add("double ").PushEffect(0, DoublePass, 22).Add("line\n\nbreak").Pop().Add(" trick")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Effect func test #6 failed: %s", tester.ErrMsg())
	}
}

func TestDrawTwineEffectsWithSpacing(t *testing.T) {
	// notice: I'd need a new effect tester for this, to check 
	//         the actual advances and so on, but I'm too lazy

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

	// (SinglePass, push/pop, no multiline, spacing, payload)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: []byte{1, 1, 2, 3, 5},
			flags: uint8(TwineTriggerPush),
		},
		TwineEffectArgs{
			Payload: []byte{1, 1, 2, 3, 5},
			flags: uint8(TwineTriggerPop),
		},
	})
	twine.Reset()
	spacing := TwineEffectSpacing{
		PrePad  : 64, // 1 pixel
		PostPad : 64, // 1 pixel
	}
	twine.Add("unknown ").PushEffectWithSpacing(0, SinglePass, spacing, 1, 1, 2, 3, 5).Add("math ").Pop().Add("sequence")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Spacing effect func test #1 failed: %s", tester.ErrMsg())
	}

	// (SinglePass, push/pop, multiline, spacing, payload)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: []byte{0, 0, 7},
			flags: uint8(TwineTriggerPush),
		},
		TwineEffectArgs{
			Payload: []byte{0, 0, 7},
			flags: uint8(TwineTriggerLineBreak),
		},
		TwineEffectArgs{
			Payload: []byte{0, 0, 7},
			flags: uint8(TwineTriggerLineStart),
		},
		TwineEffectArgs{
			Payload: []byte{0, 0, 7},
			flags: uint8(TwineTriggerPop),
		},
	})
	twine.Reset()
	spacing = TwineEffectSpacing{
		PrePad  : 128, // 2 pixels
		PostPad : 128, // 2 pixels
		LineStartPad : 64, // 1 pixel
		LineBreakPad : 64, // 1 pixel
	}
	twine.Add("no one ").PushEffectWithSpacing(0, SinglePass, spacing, 0, 0, 7).Add("cares\nthat ").Pop().Add("much")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Spacing effect func test #2 failed: %s", tester.ErrMsg())
	}
}

func TestDrawMixedTwineEffects(t *testing.T) {
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

	// (SinglePass wrapping DoublePass, no multiline)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush),
		},
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
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop),
		},
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").Pop().Add("3 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced effect func test #1 failed: %s", tester.ErrMsg())
	}

	// (DoublePass wrapping SinglePass, no multiline)
	// (soft pop test)
	tester.Init([]TwineEffectArgs{
		// measuring pass
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring,
		},
		// drawing pass
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush),
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop),
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass,
		},
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, DoublePass).Add("1 ").PushEffect(0, SinglePass).Add("2 ").Pop().Add("3 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced effect func test #2 failed: %s", tester.ErrMsg())
	}
	
	// (SinglePass wrapping DoublePass, multiline)
	tester.Init([]TwineEffectArgs{
		// single pass start
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush),
		},
		// double pass measuring
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring,
		},
		// double pass drawing
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass,
		},
		// single pass line break
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak),
		},
		// single pass second line
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart),
		},
		// double pass measuring
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring,
		},
		// double pass drawing
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass,
		},
		// single pass end
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop),
		},
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2\n3 ").Pop().Add("4 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced effect func test #3 failed: %s", tester.ErrMsg())
	}

	// (DoublePass wrapping SinglePass, multiline)
	tester.Init([]TwineEffectArgs{
		// double pass measuring
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring,
		},
		// single pass push
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagMeasuring,
		},
		// line break and reset
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring,
		},
		// first line drawing pass
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPush),
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak),
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass,
		},
		// second line pass
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagMeasuring,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring,
		},

		// second line drawing pass
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass,
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerLineStart),
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop),
		},
		TwineEffectArgs{
			Payload: nil,
			flags: uint8(TwineTriggerPop) | twineFlagDoublePass,
		},
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, DoublePass).Add("1 ").PushEffect(0, SinglePass).Add("2\n3 ").Pop().Add("4 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced effect func test #4 failed: %s", tester.ErrMsg())
	}

	// (SinglePass wrapping DoublePass wrapping SinglePass, multiline)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring }, // dpReset
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) }, // root single pass line break
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) }, // root single pass line start, drawing
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring }, // dp wrapping line start
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring }, // dp reset
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) }, // final pop for the outermost single pass
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").Pop().Add("5 ").Pop().Add("6 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced effect func test #5 failed: %s", tester.ErrMsg())
	}

	// variant of the previous using PopAll(), which should have the same triggering
	tester.ResetIndexAndErr()
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").Pop().Add("5 ").PopAll().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced effect func test #6 failed: %s", tester.ErrMsg())
	}
	
	// yet another variant with bigger scope PopAll()
	tester.ResetIndexAndErr()
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").PopAll().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced effect func test #7 failed: %s", tester.ErrMsg())
	}	
}

func TestDrawMixedTwineEffectsCounterDir(t *testing.T) {
	if testFontA == nil { t.SkipNow() }

	renderer := NewRenderer()
	renderer.SetFont(testFontA)
	renderer.Utils().SetCache8MiB()
	renderer.SetAlign(Right)

	// create tester
	var tester twineEffectTester
	var twine Twine
	target := image.NewRGBA(image.Rect(0, 0, 640, 480))
	
	// register effect func
	renderer.Twine().RegisterEffectFunc(0, tester.EffectFunc)

	// (SinglePass wrapping DoublePass, no multiline)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").Pop().Add("3 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced counter dir effect func test #1 failed: %s", tester.ErrMsg())
	}

	// (DoublePass wrapping SinglePass, no multiline)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, DoublePass).Add("1 ").PushEffect(0, SinglePass).Add("2 ").Pop().Add("3 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced counter dir effect func test #2 failed: %s", tester.ErrMsg())
	}
	
	// (SinglePass wrapping DoublePass, multiline)
	tester.Init([]TwineEffectArgs{
		// measure first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		// draw first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		// measure second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		// draw second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2\n3 ").Pop().Add("4 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced counter dir effect func test #3 failed: %s", tester.ErrMsg())
	}

	// (DoublePass wrapping SinglePass, multiline)
	tester.Init([]TwineEffectArgs{
		// measure first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring },
		// draw first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass },
		// measure second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		// draw second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, DoublePass).Add("1 ").PushEffect(0, SinglePass).Add("2\n3 ").Pop().Add("4 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced counter dir effect func test #4 failed: %s", tester.ErrMsg())
	}

	// (SinglePass wrapping DoublePass wrapping SinglePass, multiline)
	tester.Init([]TwineEffectArgs{
		// measure first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		// draw first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		// measure second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		// draw second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").Pop().Add("5 ").Pop().Add("6 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced counter dir effect func test #5 failed: %s", tester.ErrMsg())
	}

	// variant of the previous using PopAll(), which should have the same triggering
	tester.ResetIndexAndErr()
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").Pop().Add("5 ").PopAll().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced counter dir effect func test #6 failed: %s", tester.ErrMsg())
	}
	
	// yet another variant with bigger scope PopAll()
	tester.ResetIndexAndErr()
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").PopAll().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced counter dir effect func test #7 failed: %s", tester.ErrMsg())
	}
}

func TestDrawMixedTwineEffectsCentered(t *testing.T) {
	if testFontA == nil { t.SkipNow() }

	renderer := NewRenderer()
	renderer.SetFont(testFontA)
	renderer.Utils().SetCache8MiB()
	renderer.SetAlign(HorzCenter)

	// create tester
	var tester twineEffectTester
	var twine Twine
	target := image.NewRGBA(image.Rect(0, 0, 640, 480))
	
	// register effect func
	renderer.Twine().RegisterEffectFunc(0, tester.EffectFunc)

	// (SinglePass wrapping DoublePass, no multiline)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").Pop().Add("3 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced centered effect func test #1 failed: %s", tester.ErrMsg())
	}

	// (DoublePass wrapping SinglePass, no multiline)
	tester.Init([]TwineEffectArgs{
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, DoublePass).Add("1 ").PushEffect(0, SinglePass).Add("2 ").Pop().Add("3 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced centered effect func test #2 failed: %s", tester.ErrMsg())
	}
	
	// (SinglePass wrapping DoublePass, multiline)
	tester.Init([]TwineEffectArgs{
		// measure first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		// draw first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		// measure second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		// draw second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2\n3 ").Pop().Add("4 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced centered effect func test #3 failed: %s", tester.ErrMsg())
	}

	// (DoublePass wrapping SinglePass, multiline)
	tester.Init([]TwineEffectArgs{
		// measure first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring },
		// draw first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass },
		// measure second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		// draw second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, DoublePass).Add("1 ").PushEffect(0, SinglePass).Add("2\n3 ").Pop().Add("4 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced centered effect func test #4 failed: %s", tester.ErrMsg())
	}

	// (SinglePass wrapping DoublePass wrapping SinglePass, multiline)
	tester.Init([]TwineEffectArgs{
		// measure first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagMeasuring },
		// draw first line
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPush) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineBreak) },
		// measure second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass | twineFlagMeasuring },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagMeasuring },
		// draw second line
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerLineStart) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) | twineFlagDoublePass },
		TwineEffectArgs{ flags: uint8(TwineTriggerPop) },
	})
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").Pop().Add("5 ").Pop().Add("6 ").Pop().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced centered effect func test #5 failed: %s", tester.ErrMsg())
	}

	// variant of the previous using PopAll(), which should have the same triggering
	tester.ResetIndexAndErr()
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").Pop().Add("5 ").PopAll().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced centered effect func test #6 failed: %s", tester.ErrMsg())
	}
	
	// yet another variant with bigger scope PopAll()
	tester.ResetIndexAndErr()
	twine.Reset()
	twine.Add("wrap ").PushEffect(0, SinglePass).Add("1 ").PushEffect(0, DoublePass).Add("2 ").PushEffect(0, SinglePass).Add("3\n4 ").PopAll().Add("done")
	renderer.Twine().Draw(target, twine, 32, 32)
	tester.EndSequence()
	if tester.HasError() {
		t.Fatalf("Advanced centered effect func test #7 failed: %s", tester.ErrMsg())
	}
}
