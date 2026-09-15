//go:build gtxt

package etxt

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestDrawBilinear(t *testing.T) {
	if testFontA == nil {
		t.SkipNow()
	}

	newRenderer := func() (*Renderer, *image.RGBA) {
		renderer := NewRenderer()
		renderer.SetFont(testFontA)
		renderer.SetSize(18)
		renderer.SetColor(color.RGBA{255, 255, 255, 255})
		return renderer, image.NewRGBA(image.Rect(0, 0, 80, 64))
	}
	drawAt := func(x int) []byte {
		renderer, target := newRenderer()
		renderer.Draw(target, "H", x, 40)
		return target.Pix
	}
	drawBilinearAt := func(x float64) []byte {
		renderer, target := newRenderer()
		renderer.DrawBilinear(target, "H", x, 40)
		return target.Pix
	}

	// whole coordinates leave nothing to interpolate
	if !bytes.Equal(drawAt(10), drawBilinearAt(10)) {
		t.Fatal("DrawBilinear at whole coordinates differs from Draw")
	}

	// half a pixel must land halfway between both whole positions. Only one
	// glyph is drawn, neighbouring glyphs can overlap and break the comparison
	left, right := drawAt(10), drawAt(11)
	mid := make([]byte, len(left))
	for i := range mid {
		mid[i] = byte((int(left[i]) + int(right[i])) / 2)
	}
	if !similarByteSlices(mid, drawBilinearAt(10.5)) {
		t.Fatal("DrawBilinear at half a pixel isn't halfway between both whole positions")
	}
}
