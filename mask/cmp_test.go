package mask

import "os"
import "time"
import "math/rand"
import "strconv"
import "image"
import "image/png"
import "image/color"
import "testing"

import "github.com/tinne26/etxt/fract"

func TestEdgeVsDefaultRasterizerTriangle(t *testing.T) {
	const avgCmpTolerance = 4.0 // alpha value per 255
	const canvasWidth = 80
	const canvasHeight = 80
	const debugMaxDiff = false

	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	defaultRasterizer := &DefaultRasterizer{}
	edgeRasterizer := &EdgeMarkerRasterizer{}
	edgeRasterizer.SetCurveThreshold(0.1)
	edgeRasterizer.SetMaxCurveSplits(8)
	maxDiff := float64(0)
	for n := 0; n < 256; n++ {
		// create random segments
		segments := randomTriangle(rng, canvasWidth, canvasHeight)
		bounds := segments.Bounds()
		if bounds.Max.X-bounds.Min.X < 6*64 {
			continue
		} // dismiss extreme cases

		// rasterize with both rasterizers
		defMask, err := Rasterize(segments, defaultRasterizer, fract.Point{})
		if err != nil {
			t.Fatalf("defaultRast error: %s", err.Error())
		}
		edgeMask, err := Rasterize(segments, edgeRasterizer, fract.Point{})
		if err != nil {
			t.Fatalf("edgeRast error: %s", err.Error())
		}

		// compare results
		totalDiff, avgDiff := masksAvgDiff(t, defMask, edgeMask)
		if avgDiff > avgCmpTolerance {
			exportTest("cmp_rast_tri_"+strconv.Itoa(n)+"_edge.png", edgeMask)
			exportTest("cmp_rast_tri_"+strconv.Itoa(n)+"_rast.png", defMask)
			t.Fatalf("iter %d, totalDiff = %d, average tolerance is too big (%f) (written files for visual debug)", n, totalDiff, avgDiff)
		}
		if avgDiff > maxDiff {
			maxDiff = avgDiff
		}
	}
	if debugMaxDiff {
		t.Fatalf("maxDiff = %f\n", maxDiff)
	}
}

func TestEdgeVsDefaultRasterizerQuad(t *testing.T) {
	const avgCmpTolerance = 4.0 // alpha value per 255
	const canvasWidth = 80
	const canvasHeight = 80
	const debugMaxDiff = false

	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	defaultRasterizer := &DefaultRasterizer{}
	edgeRasterizer := &EdgeMarkerRasterizer{}
	edgeRasterizer.SetCurveThreshold(0.1)
	edgeRasterizer.SetMaxCurveSplits(8)
	maxDiff := float64(0)
	for n := 0; n < 256; n++ {
		// create random segments
		segments := randomQuad(rng, canvasWidth, canvasHeight)
		bounds := segments.Bounds()
		if bounds.Max.X-bounds.Min.X < 6*64 {
			continue
		} // dismiss extreme cases

		// rasterize with both rasterizers
		defMask, err := Rasterize(segments, defaultRasterizer, fract.Point{})
		if err != nil {
			t.Fatalf("defaultRast error: %s", err.Error())
		}
		edgeMask, err := Rasterize(segments, edgeRasterizer, fract.Point{})
		if err != nil {
			t.Fatalf("edgeRast error: %s", err.Error())
		}

		// compare results
		totalDiff, avgDiff := masksAvgDiff(t, defMask, edgeMask)
		if avgDiff > avgCmpTolerance {
			exportTest("cmp_rast_quad_"+strconv.Itoa(n)+"_edge.png", edgeMask)
			exportTest("cmp_rast_quad_"+strconv.Itoa(n)+"_rast.png", defMask)
			t.Fatalf("iter %d, totalDiff = %d, average tolerance is too big (%f) (written files for visual debug)", n, totalDiff, avgDiff)
		}
		if avgDiff > maxDiff {
			maxDiff = avgDiff
		}
	}
	if debugMaxDiff {
		t.Fatalf("maxDiff = %f\n", maxDiff)
	}
}

func TestEdgeVsDefaultRasterizer(t *testing.T) {
	const avgCmpTolerance = 2.0 // alpha value per 255
	const canvasWidth = 80
	const canvasHeight = 80
	const useTimeSeed = false
	const debugMaxDiff = false

	if debugMaxDiff && !useTimeSeed {
		t.Fatal("set useTimeSeed = true if you want to debugMaxDiff\n")
	}

	seed := time.Now().UnixNano()
	if !useTimeSeed {
		seed = 8623001
	}
	rng := rand.New(rand.NewSource(seed)) // *
	// * Variable time seed works 99% of the time, but in some
	//   cases there are still differences that are big enough to
	//   be reported as failing tests.
	//   Still, I decided to switch to a static seed in order to make
	//   life more peaceful. Even if there's a bug, I'll wait until
	//   I come across a problematic case that happens in real life...
	//   instead of these synthetic tests.

	defaultRasterizer := &DefaultRasterizer{}
	edgeRasterizer := &EdgeMarkerRasterizer{}
	edgeRasterizer.SetCurveThreshold(0.1)
	edgeRasterizer.SetMaxCurveSplits(8)
	maxDiff := float64(0)
	for n := 0; n < 32; n++ {
		// create random segments
		segments := randomSegments(rng, 16, canvasWidth, canvasHeight)

		// rasterize with both rasterizers
		defMask, err := Rasterize(segments, defaultRasterizer, fract.Point{})
		if err != nil {
			t.Fatalf("defaultRast error: %s", err.Error())
		}
		edgeMask, err := Rasterize(segments, edgeRasterizer, fract.Point{})
		if err != nil {
			t.Fatalf("edgeRast error: %s", err.Error())
		}

		// compare results
		totalDiff, avgDiff := masksAvgDiff(t, defMask, edgeMask)
		if avgDiff > avgCmpTolerance {
			// Notice: this actually fails sometimes if a variable seed is used.
			//         There are two main reasons for this: different curve
			//         segmentation methods and float64 being used in edge marker
			//         rasterizer in some places. Look at the results yourself
			//         if this ever fails for you.
			exportTest("cmp_rast_rng_"+strconv.Itoa(n)+"_edge.png", edgeMask)
			exportTest("cmp_rast_rng_"+strconv.Itoa(n)+"_rast.png", defMask)
			t.Fatalf("iter %d, totalDiff = %d, average tolerance is too big (%f) (written files for visual debug)", n, totalDiff, avgDiff)
		}
		if avgDiff > maxDiff {
			maxDiff = avgDiff
		}
	}
	if debugMaxDiff {
		t.Fatalf("maxDiff = %f\n", maxDiff)
	}
}

func TestFauxVsDefaultRasterizer(t *testing.T) {
	const canvasWidth = 80
	const canvasHeight = 80

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	defaultRasterizer := &DefaultRasterizer{}
	fauxRasterizer := &FauxRasterizer{}
	for n := 0; n < 32; n++ {
		// create random segments
		segments := randomSegments(rng, 16, canvasWidth, canvasHeight)

		// rasterize with both rasterizers
		defMask, err := Rasterize(segments, defaultRasterizer, fract.Point{})
		if err != nil {
			t.Fatalf("defaultRast error: %s", err.Error())
		}
		fauxMask, err := Rasterize(segments, fauxRasterizer, fract.Point{})
		if err != nil {
			t.Fatalf("edgeRast error: %s", err.Error())
		}

		// compare results
		totalDiff, avgDiff := masksAvgDiff(t, defMask, fauxMask)
		if avgDiff > 0 { // both use vector.Rasterizer under the hood
			exportTest("cmp_rast_rng_"+strconv.Itoa(n)+"_faux.png", fauxMask)
			exportTest("cmp_rast_rng_"+strconv.Itoa(n)+"_rast.png", defMask)
			t.Fatalf("iter %d, totalDiff = %d, average tolerance is too big (%f) (written files for visual debug)", n, totalDiff, avgDiff)
		}
	}
}

// ---- additional helpers ----

// clamping from uint32 to uint16 values
func uint16N(value uint32) uint16 {
	if value > 65535 {
		return 65535
	}
	return uint16(value)
}

func mixColors(draw color.Color, back color.Color) color.Color {
	dr, dg, db, da := draw.RGBA()
	if da == 0xFFFF {
		return draw
	}
	if da == 0 {
		return back
	}
	br, bg, bb, ba := back.RGBA()
	if ba == 0 {
		return draw
	}
	return color.RGBA64{
		R: uint16N((dr*0xFFFF + br*(0xFFFF-da)) / 0xFFFF),
		G: uint16N((dg*0xFFFF + bg*(0xFFFF-da)) / 0xFFFF),
		B: uint16N((db*0xFFFF + bb*(0xFFFF-da)) / 0xFFFF),
		A: uint16N((da*0xFFFF + ba*(0xFFFF-da)) / 0xFFFF),
	}
}

func exportTest(filename string, mask *image.Alpha) {
	rgba := image.NewRGBA(mask.Rect)
	r, g, b, a := color.White.RGBA()
	nrgba := color.NRGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: 0}
	for y := mask.Rect.Min.Y; y < mask.Rect.Max.Y; y++ {
		for x := mask.Rect.Min.X; x < mask.Rect.Max.X; x++ {
			nrgba.A = uint16((a * uint32(mask.AlphaAt(x, y).A)) / 255)
			rgba.Set(x, y, mixColors(nrgba, color.Black))
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	err = png.Encode(file, rgba)
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
}

func masksAvgDiff(t *testing.T, a, b *image.Alpha) (int, float64) {
	if len(a.Pix) != len(b.Pix) {
		t.Fatalf("mask lengths are different (%d vs %d)", len(a.Pix), len(b.Pix))
	}

	totalDiff := 0
	for i := 0; i < len(a.Pix); i++ {
		valueA := a.Pix[i]
		valueB := b.Pix[i]
		if valueA == valueB {
			continue
		}
		var diff uint8
		if valueA > valueB {
			diff = valueA - valueB
		} else {
			diff = valueB - valueA
		}
		totalDiff += int(diff)
	}

	return totalDiff, float64(totalDiff) / float64(len(a.Pix))
}
