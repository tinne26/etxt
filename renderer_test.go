//go:build gtxt

package etxt

import "os"
import "image"
import "image/color"
import "image/png"
import "log"

import "testing"

import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

import "github.com/tinne26/etxt/emask"
import "github.com/tinne26/etxt/esizer"
import "github.com/tinne26/etxt/efixed"

func TestSetGet(t *testing.T) {
	// mostly tests the renderer default values
	rast := emask.FauxRasterizer{}
	renderer := NewRenderer(&rast)
	vAlign, hAlign := renderer.GetAlign()
	if vAlign != Baseline { t.Fatalf("expected Baseline, got %d", vAlign) }
	if hAlign != Left     { t.Fatalf("expected Left, got %d", hAlign) }

	handler := renderer.GetCacheHandler()
	if handler != nil { t.Fatalf("expected nil cache handler") }

	rgba, isRgba := renderer.GetColor().(color.RGBA)
	if !isRgba { t.Fatal("expected rgba color") }
	if rgba.R != 255 || rgba.G != 255 || rgba.B != 255 || rgba.A != 255 {
		t.Fatalf("expected white")
	}

	font := renderer.GetFont()
	if font != nil { t.Fatal("expected nil font") }

	renderer.SetLineHeight(10)
	renderer.SetLineSpacing(2)
	advance := renderer.GetLineAdvance()
	if advance != (20 << 6) { t.Fatalf("expected advance = 20, got %f", float64(advance)/64) }
	renderer.SetLineHeightAuto()

	if renderer.GetRasterizer() != &rast { t.Fatal("what") }

	sizePx := renderer.GetSizePxFract()
	if sizePx != 16 << 6 { t.Fatalf("expected size = 16, got %f", float64(sizePx)/64) }
	renderer.SetSizePxFract(17 << 6)
	sizePx  = renderer.GetSizePxFract()
	if sizePx != 17 << 6 { t.Fatalf("expected size = 17, got %f", float64(sizePx)/64) }

	sizer := renderer.GetSizer()
	_, isDefaultSizer := sizer.(*esizer.DefaultSizer)
	if !isDefaultSizer { t.Fatal("expected DefaultSizer") }

	renderer.SetVertAlign(YCenter)
	renderer.SetHorzAlign(XCenter)
	vAlign, hAlign = renderer.GetAlign()
	if vAlign != YCenter { t.Fatalf("expected YCenter, got %d", vAlign) }
	if hAlign != XCenter { t.Fatalf("expected XCenter, got %d", hAlign) }
}

func TestSelectionRect(t *testing.T) {
	renderer := NewStdRenderer()
	renderer.SetFont(testFont)
	renderer.SetCacheHandler(NewDefaultCache(1024).NewHandler())
	renderer.SetDirection(RightToLeft)

	rect := renderer.SelectionRect("hey ho")
	if rect.Width.Ceil() < 32 {
		t.Fatalf("expected Width.Ceil to be at least 32, but got %d", rect.Width.Ceil())
	}
	if rect.Width.Ceil() > 128 {
		t.Fatalf("expected Width.Ceil to be below 128, but got %d", rect.Width.Ceil())
	}
	if rect.Height.Ceil() < 8 {
		t.Fatalf("expected Height.Ceil to be at least 8, but got %d", rect.Height.Ceil())
	}
	imgRect := rect.ImageRect()
	rect2 := renderer.SelectionRect("hey ho hey ho")
	if !imgRect.In(rect2.ImageRect()) { t.Fatal("inconsistent rects") }

	testGlyphs := make([]GlyphIndex, 0, len("hey ho"))
	var buffer sfnt.Buffer
	for _, codePoint := range "hey ho" {
		index, err := testFont.GlyphIndex(&buffer, codePoint)
		if err != nil { panic(err) }
		if index == 0 { panic(err) }
		testGlyphs = append(testGlyphs, index)
	}

	renderer.SetLineSpacing(0)
	imgRect3 := renderer.SelectionRect("hey ho\nhey ho").ImageRect()
	if !imgRect3.Eq(imgRect) {
		t.Fatalf("line spacing 0 failed (%v vs %v)", imgRect, imgRect3)
	}

	// test line breaks and other edge cases
	renderer.SetLineSpacing(1) // restore spacing
	renderer.SetQuantizationMode(QuantizeNone) // prevent vertical quantization adjustments
	renderer.SetDirection(LeftToRight)
	rect = renderer.SelectionRect("")
	if rect.Width != 0 || rect.Height != 0 {
		t.Fatalf("expected Width and Height to be 0, but got ~%d", rect.Width.Ceil())
	}
	rect = renderer.SelectionRect("MMMMM")
	baseHeight := efixed.ToFloat64(rect.Height)
	rect = renderer.SelectionRect(" ")
	checkHeight := efixed.ToFloat64(rect.Height)
	if checkHeight != baseHeight {
		t.Fatalf("expected Height to be %f, but got %f", baseHeight, checkHeight)
	}

	rect = renderer.SelectionRect("MMM\n")
	heightA := efixed.ToFloat64(rect.Height)
	rect = renderer.SelectionRect("\nMMM")
	heightB := efixed.ToFloat64(rect.Height)
	if heightA != heightB {
		t.Fatalf("expected heightA (%f) == heightB (%f)", heightA, heightB)
	}
	if heightA == baseHeight {
		t.Fatalf("expected heightA to be different from baseHeight (%f), but got %f", baseHeight, heightA)
	}
	if heightA != baseHeight*2 {
		t.Fatalf("baseHeight = %f, heightA = %f", baseHeight, heightA)
		t.Fatalf("expected heightA to be baseHeight*2 (%f), but got %f", baseHeight*2, heightA)
	}
	rect = renderer.SelectionRect("\n")
	checkHeight = efixed.ToFloat64(rect.Height)
	if checkHeight != baseHeight {
		t.Fatalf("expected \\n Height to be %f, but got %f", baseHeight, checkHeight)
	}
	rect = renderer.SelectionRect("\n\n")
	checkHeight = efixed.ToFloat64(rect.Height)
	if checkHeight != baseHeight*2 {
		t.Fatalf("expected \\n\\n Height to be %f, but got %f", baseHeight*2, checkHeight)
	}

	rect = renderer.SelectionRect("\n\n\n")
	heightC := efixed.ToFloat64(rect.Height)
	rect = renderer.SelectionRect("\n \n")
	heightD := efixed.ToFloat64(rect.Height)
	if heightC != heightD {
		t.Fatalf("expected heightC (%f) == heightD (%f)", heightC, heightD)
	}
}

// the real consistency test
func TestStringVsGlyph(t *testing.T) {
	renderer := NewStdRenderer()
	renderer.SetSizePx(16)
	renderer.SetFont(testFont)
	renderer.SetQuantizationMode(QuantizeFull)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	alignPairs := []struct{ vert VertAlign; horz HorzAlign }{
		{vert: Baseline, horz: Left}, {vert: YCenter, horz: XCenter},
		{vert: Top, horz: Right}, {vert: Bottom, horz: Left},
	}
	quantModes := []QuantizationMode{QuantizeFull, QuantizeVert, QuantizeNone}

	testText := "for lack of better words"
	var buffer sfnt.Buffer

	missing, err := GetMissingRunes(testFont, testText)
	if err != nil { panic(err) }
	if len(missing) > 0 { panic("missing runes to test") }

	// get text as glyphs
	testGlyphs := make([]GlyphIndex, 0, len(testText))
	for _, codePoint := range testText {
		index, err := testFont.GlyphIndex(&buffer, codePoint)
		if err != nil {
			t.Fatalf("Unexpected error on testFont.GlyphIndex: " + err.Error())
		}
		if index == 0 {
			t.Fatalf("testFont.GlyphIndex missing rune '" + string(codePoint) + "'")
		}
		testGlyphs = append(testGlyphs, index)
	}

	// compute text size
	rect := renderer.SelectionRect(testText) // fully quantized
	for _, textDir := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(textDir)
		for _, quantMode := range quantModes {
			renderer.SetQuantizationMode(quantMode)
			testRect := renderer.SelectionRect(testText)
			for _, alignPair := range alignPairs {
				renderer.SetAlign(alignPair.vert, alignPair.horz)
				txtRect := renderer.SelectionRect(testText)
				if txtRect.Width != testRect.Width || txtRect.Height != testRect.Height {
					t.Fatalf("selection rect mismatch between aligns")
				}
				glyphsRect := renderer.SelectionRectGlyphs(testGlyphs)
				if glyphsRect.Width != testRect.Width || glyphsRect.Height != testRect.Height {
					t.Fatalf("selection rect mismatch between glyphs and text")
				}
			}
		}
	}

	// create target image and fill it with white
	w, h := rect.Width.Ceil()*2 + 8, rect.Height.Ceil()*2 + 8
	outImageA := image.NewRGBA(image.Rect(0, 0, w, h))
	outImageB := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h*4; i++ { outImageA.Pix[i] = 255 }
	for i := 0; i < w*h*4; i++ { outImageB.Pix[i] = 255 }

	// draw and compare results between glyphs and text
	for _, textDir := range []Direction{LeftToRight, RightToLeft} {
		renderer.SetDirection(textDir)
		for _, quantMode := range quantModes {
			renderer.SetQuantizationMode(quantMode)
			for _, alignPair := range alignPairs {
				renderer.SetAlign(alignPair.vert, alignPair.horz)
				renderer.SetTarget(outImageA)
				dotA := renderer.Draw(testText, w/2, h/2)
				renderer.SetTarget(outImageB)
				dotB := drawGlyphs(renderer, testGlyphs, w/2, h/2)
				for i := 0; i < w*h*4; i++ {
					if outImageA.Pix[i] != outImageB.Pix[i] {
						what := "drawing mismatch between glyphs and text (quantMode "
						what += "= %d, align pair = %d / %d)"
						t.Fatalf(what, quantMode, alignPair.vert, alignPair.horz)
					}
				}

				// compare returned dots
				if dotA.X != dotB.X || dotA.Y != dotB.Y {
					what := "mismatch in the dots returned by Draw/DrawGlyphs (quantMode "
					what += "= %d, align pair = %d / %d): %v vs %v"
					t.Fatalf(what, quantMode, alignPair.vert, alignPair.horz, dotA, dotB)
				}

				// clear images
				for i := 0; i < w*h*4; i++ { outImageA.Pix[i] = 255 }
				for i := 0; i < w*h*4; i++ { outImageB.Pix[i] = 255 }
			}
		}
	}
}

func drawGlyphs(renderer *Renderer, glyphIndices []GlyphIndex, x, y int) fixed.Point26_6 {
	return renderer.TraverseGlyphs(glyphIndices, fixed.P(x, y),
		func(dot fixed.Point26_6, glyphIndex GlyphIndex) {
				mask := renderer.LoadGlyphMask(glyphIndex, dot)
				renderer.DefaultDrawFunc(dot, mask, glyphIndex)
		})
}

func TestDrawCached(t *testing.T) {
	renderer := NewStdRenderer()
	renderer.SetFont(testFont)
	renderer.SetCacheHandler(NewDefaultCache(1024).NewHandler())
	target := image.NewAlpha(image.Rect(0, 0, 64, 64))
	renderer.SetTarget(target)
	renderer.Draw("dumb test", 0, 0)
	renderer.Draw("dumb test", 0, 0)
	renderer.SetSizePx(18)
	renderer.Draw("dumb test", 0, 0)
}

func TestGtxtMixModes(t *testing.T) {
	target := image.NewRGBA(image.Rect(0, 0, 64, 64))
	renderer := NewStdRenderer()
	renderer.SetFont(testFont)
	renderer.SetSizePx(24)
	renderer.SetTarget(target)

	// replace mode
	for i, _ := range target.Pix { target.Pix[i] = 255 }
	renderer.SetMixMode(MixReplace)
	renderer.Draw("O", 32, 32)

	ok := false
	for i := 0; i < len(target.Pix); i += 4 {
		alpha := target.Pix[i + 3]
		if alpha == 0 { ok = true }
		if target.Pix[i + 0] != alpha { t.Fatalf("%d, %d, %d", i, alpha, target.Pix[i + 0]) }
		if target.Pix[i + 1] != alpha { t.Fatalf("%d, %d, %d", i, alpha, target.Pix[i + 1]) }
		if target.Pix[i + 2] != alpha { t.Fatalf("%d, %d, %d", i, alpha, target.Pix[i + 2]) }
	}
	if !ok { t.Fatal("expected some transparent region, but didn't find it") }

	// mix cut mode
	renderer.SetMixMode(MixCut)
	renderer.Draw("O", 32, 32)
	for i := 0; i < len(target.Pix); i += 4 {
		alpha := target.Pix[i + 3]
		if alpha != 0 && alpha != 255 {
			t.Fatalf("unexpected alpha %d at %d", alpha, i)
		}
	}

	// sub mode
	for i, _ := range target.Pix { target.Pix[i] = 255 }
	renderer.SetMixMode(MixSub)
	renderer.SetColor(color.RGBA{255, 0, 255, 255})
	renderer.Draw("O", 32, 32)
	ok = false
	for i := 0; i < len(target.Pix); i += 4 {
		if target.Pix[i + 1] == 255 && target.Pix[i + 3] == 255 &&
		   target.Pix[i + 0] == 0   && target.Pix[i + 2] == 0 {
				ok = true // pure green found
		}
	}
	if !ok { t.Fatal("failed to find green") }

	renderer.SetMixMode(MixMultiply)
	renderer.SetColor(color.RGBA{0, 0, 0, 255})
	renderer.Draw("O", 32, 32)
	for i := 0; i < len(target.Pix); i += 4 {
		alpha := target.Pix[i + 3]
		if alpha != 255 { t.Fatalf("unexpected alpha %d at %d", alpha, i) }
		if target.Pix[i + 0] != target.Pix[i + 2] || target.Pix[i + 1] < target.Pix[i + 2] {
			t.Fatalf("bad color")
		}
	}

	// add mode
	for i := 0; i < len(target.Pix); i += 4 {
		target.Pix[i + 0] = 255
		target.Pix[i + 1] = 0
		target.Pix[i + 2] = 0
		target.Pix[i + 3] = 255
	}
	renderer.SetMixMode(MixAdd)
	renderer.SetColor(color.RGBA{0, 0, 255, 255})
	renderer.Draw("O", 32, 32)
	ok = false
	for i := 0; i < len(target.Pix); i += 4 {
		if target.Pix[i + 1] !=   0 { t.Fatal("green must be 0")   }
		if target.Pix[i + 3] != 255 { t.Fatal("alpha must be 255") }
		if target.Pix[i] == 255 && target.Pix[i + 2] == 255 { ok = true }
	}
	if !ok { t.Fatal("failed to find pure magenta") }

	// fifty-fifty mode
	for i := 0; i < len(target.Pix); i += 4 {
		target.Pix[i + 0] = 255
		target.Pix[i + 1] = 0
		target.Pix[i + 2] = 0
		target.Pix[i + 3] = 255
	}
	renderer.SetMixMode(MixFiftyFifty)
	renderer.SetColor(color.RGBA{255, 0, 255, 255})
	renderer.Draw("O", 32, 32)
	for i := 0; i < len(target.Pix); i += 4 {
		if target.Pix[i + 1] !=   0 { t.Fatal("green must be 0")   }
		if target.Pix[i + 3] != 255 { t.Fatal("alpha must be 255") }
		if target.Pix[i + 0] != 255 { t.Fatal("red must be 255") }
		if target.Pix[i + 2]  > 128 { t.Fatalf("blue over 128 %d", target.Pix[i + 2]) }
	}
}

func TestAlignBound(t *testing.T) {
	renderer := NewStdRenderer()
	renderer.SetFont(testFont)
	renderer.SetAlign(YCenter, XCenter)
	horzPadder := &esizer.HorzPaddingSizer{}
	renderer.SetSizer(horzPadder)

	for size := 7; size < 73; size += 3 {
		renderer.SetSizePx(size)
		prevWidth := fixed.Int26_6(0)
		for i := 0; i < 64; i += 3 {
			const text = "abcdefghijkl - mnopq0123456789"
			horzPadder.SetHorzPaddingFract(fixed.Int26_6(i))

			renderer.SetQuantizationMode(QuantizeFull)
			rect1 := renderer.SelectionRect(text)
			renderer.SetQuantizationMode(QuantizeVert)
			rect2 := renderer.SelectionRect(text)
			renderer.SetQuantizationMode(QuantizeNone)
			rect3 := renderer.SelectionRect(text)

			if rect1.Height != rect2.Height {
				t.Fatalf("SelectionRect.Height different for QuantizeFull and QuantizeVert (%d vs %d)", rect1.Height, rect2.Height)
			}
			if rect2.Width != rect3.Width {
				t.Fatalf("SelectionRect.Width different for QuantizeVert and QuantizeNone (%d vs %d)", rect2.Width, rect3.Width)
			}
			if rect3.Width <= prevWidth {
				t.Fatalf("SelectionRect.Width didn't increase")
			}
			prevWidth = rect3.Width

			// if rect1.Width == rect2.Width { // uncommon but this can happen legitimately
			// 	t.Fatalf("SelectionRect.Width uncommon match for QuantizeFull and QuantizeVert (%d vs %d)", rect1.Width, rect2.Width)
			// }
		}
	}
}

func debugExport(name string, img image.Image) {
	file, err := os.Create(name)
	if err != nil { log.Fatal(err) }
	err = png.Encode(file, img)
	if err != nil { log.Fatal(err) }
	err = file.Close()
	if err != nil { log.Fatal(err) }
}
