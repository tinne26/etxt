package etxt

import "fmt"

import "testing"

import "github.com/tinne26/etxt/fract"

func TestMeasure(t *testing.T) {
	if testFontA == nil { t.SkipNow() }

	renderer := NewRenderer()
	renderer.SetFont(testFontA)
	renderer.Utils().SetCache8MiB()

	testMeasureBasics(t, renderer, func(r *Renderer, str string) fract.Rect {
		return r.Measure(str)
	})
}

func TestMeasureWithWrap(t *testing.T) {
	if testFontA == nil { t.SkipNow() }

	renderer := NewRenderer()
	renderer.SetFont(testFontA)
	renderer.Utils().SetCache8MiB()

	testMeasureBasics(t, renderer, func(r *Renderer, str string) fract.Rect {
		fmt.Printf("processing: '%s'\n", str)
		return r.MeasureWithWrap(str, 9999)
	})
}

func testMeasureBasics(t *testing.T, renderer *Renderer, fn func(*Renderer, string) fract.Rect) {
	for _, qt := range []fract.Unit{ QtFull, QtHalf, Qt4th, QtNone } {
		for _, align := range []Align{ Baseline | Left, Baseline | Right, Center } {
			for _, dir := range []Direction{ LeftToRight, RightToLeft } {
				fmt.Printf("config: qt = %d, align = %s, dir = %s\n", qt, align.String(), dir.String())

				// configure renderer with current params
				renderer.Fract().SetHorzQuantization(qt)
				renderer.SetAlign(align)
				renderer.Complex().SetDirection(dir)

				// check zero origin
				if !fn(renderer, "\n ya\n \n").HasZeroOrigin() {
					t.Fatal("measure rects should always have zero origin")
				}
				zw, zh := fn(renderer, "").Size()
				if zw != 0 || zh != 0 {
					t.Fatal("expected zero with and height")
				}

				// consistency tests
				w1, h1 := fn(renderer, "hey h").Size()
				w2, h2 := fn(renderer, "hey ho").Size()
				w3, h3 := fn(renderer, "hey hoo").Size()
				w4,  _ := fn(renderer, "hey ho.hey ho").Size()
				if w3 >= w1*2 {
					t.Fatalf("expected w3 < w1*2, but got w3 = %d, w1 = %d", w3, w1)
				}
				if w1 >= w2 {
					t.Fatalf("expected w1 < w2, but got w2 = %d, w1 = %d", w2, w1)
				}
				if w3 <= w2 {
					t.Fatalf("expected w3 > w2, but got w3 = %d, w2 = %d", w3, w2)
				}
				if h1 != h2 || h2 != h3 {
					t.Fatalf("inconsistent heights (%d, %d, %d)", h1, h2, h3)
				}
				if w4 <= w2*2 {
					t.Fatalf("expected w4 > w2*2, but got w4 = %d, w2 = %d", w4, w2)
				}

				// line break and spacing tests
				h5 := fn(renderer, "\n").Height()
				if h5 != h1 {
					t.Fatal("expected line break height to match regular line")
				}
				h6 := fn(renderer, "\n ").Height()
				if h6 <= h5 {
					t.Fatal("expected content to exceed line break's height")
				}

				hs1 := fn(renderer, "A").Height()
				hs2 := fn(renderer, " ").Height()
				if hs1 != hs2 { t.Fatal("expected same height") }
				hs1 = fn(renderer, "A\n\nA").Height()
				hs2 = fn(renderer, "    \n\n      ").Height()
				if hs1 != hs2 { t.Fatal("expected same height") }
			}
		}
	}

	// direction symmetry check (only reliable when quantization is fully disabled)
	renderer.Fract().SetHorzQuantization(QtNone)
	for _, align := range []Align{ Baseline | Left, Baseline | Right, Center } {	
		renderer.SetAlign(align)

		renderer.Complex().SetDirection(LeftToRight)
		w1, h1 := fn(renderer, "\nABCD\n").Size()
		renderer.Complex().SetDirection(RightToLeft)
		w2, h2 := fn(renderer, "\nDCBA\n").Size()
		if w1 != w2 || h1 != h2 {
			t.Fatalf("expected w1, h1 == w2, h2, but got %d, %d != %d, %d", w1, h1, w2, h2)
		}

		renderer.Complex().SetDirection(LeftToRight)
		w1, h1 = fn(renderer, "hello world").Size()
		renderer.Complex().SetDirection(RightToLeft)
		w2, h2 = fn(renderer, "dlrow olleh").Size()
		if w1 != w2 || h1 != h2 {
			t.Fatalf("expected w1, h1 == w2, h2, but got %d, %d != %d, %d", w1, h1, w2, h2)
		}
	}
}
