//go:build test

package emask

import "image/color"
import "testing"

import "golang.org/x/image/math/fixed"

func TestShape(t *testing.T) {
	shape := NewShape(5)

	shape.MoveTo(0, 50)
	shape.CubeTo(-25, -25, -25, 25, 0, -50)
	shape.QuadTo(-50, 0, 0, 50)

	shape.MoveTo(0, 50)
	shape.CubeTo(25, -25, 25, 25, 0, -50)
	shape.QuadTo(50, 0, 0, 50)

	shape.MoveTo( 0, -50)
	shape.LineTo( 0,  50)
	shape.LineTo(50,  50)
	shape.LineTo(50, -50)
	shape.LineTo( 0, -50)

	mask, err := Rasterize(shape.Segments(), &DefaultRasterizer{}, fixed.P(0, 0))
	if err != nil { panic(err) }

	gotSize := mask.Rect.Dx()
	if gotSize != 100 {
		t.Fatalf("expected mask rect.Dx == 100, got %d", gotSize)
	}
	gotSize  = mask.Rect.Dy()
	if gotSize != 100 {
		t.Fatalf("expected mask rect.Dx == 100, got %d", gotSize)
	}

	posSum := 0
	negSum := 0
	for _, value := range mask.Pix {
		intValue := int(value)
		posSum += intValue
		negSum += 255 - intValue
	}
	halfSum := 255*100*50

	posSumDist := halfSum - posSum
	if posSumDist < -halfSum/1000 || posSumDist > halfSum/1000 {
		t.Fatalf("expected posSum (%d) around %d", posSum, halfSum)
	}

	negSumDist := halfSum - negSum
	if negSumDist < -halfSum/1000 || negSumDist > halfSum/1000 {
		t.Fatalf("expected negSum (%d) around %d", negSum, halfSum)
	}

	rgbSum := 0
	img := shape.Paint(color.White, color.Black)
	for i := 0; i < len(img.Pix); i += 4 {
		rgbSum += int(img.Pix[i]) + int(img.Pix[i + 1]) + int(img.Pix[i + 2])
	}
	avgColorChanValue := rgbSum/(100*100*3)
	if avgColorChanValue < 127 || avgColorChanValue > 129 {
		t.Fatalf("expected avg color (%d) around 128", avgColorChanValue)
	}

	shape.Reset()
	if len(shape.Segments()) != 0 {
		t.Fatal("expected zero segments after reset")
	}
}
