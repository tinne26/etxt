//go:build gtxt

package main

import "os"
import "log"
import "fmt"
import "strconv"
import "strings"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/font"

// Must be compiled with '-tags gtxt'

// This program prints info about the given font directly to standard
// output. Mostly metrics. This is less of an example and more a useful
// tool for debugging and checking specific font metrics.

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be examined\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil { log.Fatal(err) }

	// start collecting and printing some basic font info
	var buffer sfnt.Buffer
	fmt.Printf("# %s\n", fontName)

	info, err := font.GetIdentifier(sfntFont)
	if err != nil {
		if err != font.ErrNotFound { log.Fatal(err) }
		info = "(not found)"
	}
	fmt.Printf("Identifier  : %s\n", info)
	fmt.Printf("Num. Glyphs : %d\n\n", sfntFont.NumGlyphs())

	info, err = font.GetFamily(sfntFont)
	if err != nil {
		if err != font.ErrNotFound { log.Fatal(err) }
		info = "(not found)"
	}
	fmt.Printf("Family  : %s\n", info)

	info, err = font.GetSubfamily(sfntFont)
	if err != nil {
		if err != font.ErrNotFound { log.Fatal(err) }
		info = "(not found)"
	}
	fmt.Printf("Style   : %s\n", info)

	postTable := sfntFont.PostTable()
	if postTable.ItalicAngle != 0 {
		fmt.Printf("Slant   : %s degrees\n", minFloatFmt2(postTable.ItalicAngle))
	}
	if postTable.IsFixedPitch {
		fmt.Print("Spacing : Monospaced\n")
	} else {
		fmt.Print("Spacing : Proportional\n")
	}
	fmt.Printf("\nEm Square   : %d units\n", sfntFont.UnitsPerEm())

	const NoHinting = 0
	unitSize := fixed.Int26_6(sfntFont.UnitsPerEm())
	metrics, err := sfntFont.Metrics(&buffer, unitSize, NoHinting)
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font Height : %4d units\n", metrics.Height)
	fmt.Printf("Ascent      : %4d units\n", metrics.Ascent)
	fmt.Printf("Descent     : %4d units\n", metrics.Descent)
	fmt.Printf("Line Gap    : %4d units\n", metrics.Height - metrics.Ascent - metrics.Descent)
	relCapHeight := float64(metrics.CapHeight)/float64(unitSize)
	relXHeight   := float64(metrics.XHeight)/float64(unitSize)
	fmt.Printf("Cap. Height : %4d units (%s%% em height)\n", metrics.CapHeight, minFloatFmt2(100*relCapHeight))
	fmt.Printf("XHeight     : %4d units (%s%% em height)\n", metrics.XHeight, minFloatFmt2(100*relXHeight))
	ascUn, ascent, err := RuneAscent(sfntFont, 'T', &buffer)
	if err == nil { fmt.Printf("   Actual 'T' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = RuneAscent(sfntFont, 'A', &buffer)
	if err == nil { fmt.Printf("   Actual 'A' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = RuneAscent(sfntFont, 'O', &buffer)
	if err == nil { fmt.Printf("   Actual 'O' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = RuneAscent(sfntFont, 'x', &buffer)
	if err == nil { fmt.Printf("   Actual 'x' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = RuneAscent(sfntFont, 'a', &buffer)
	if err == nil { fmt.Printf("   Actual 'a' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = RuneAscent(sfntFont, 'r', &buffer)
	if err == nil { fmt.Printf("   Actual 'r' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }

	var minKern, maxKern fixed.Int26_6
	var minKernSet bool
	kernPairs := 0
	kernEvalCount := sfntFont.NumGlyphs()
	if kernEvalCount > 1000 { kernEvalCount = 1000 }
	for i := 0; i < kernEvalCount; i++ {
		for j := 0; j < kernEvalCount; j++ {
			kern, err := sfntFont.Kern(&buffer, sfnt.GlyphIndex(i), sfnt.GlyphIndex(j), unitSize, NoHinting)
			if err != nil {
				if err == sfnt.ErrNotFound { continue }
				log.Fatal(err)
			}
			if !minKernSet { minKern = kern ; minKernSet = true }
			if kern < minKern { minKern = kern }
			if kern > maxKern { maxKern = kern }
			kernPairs += 1
		}
	}
	if minKern == 0 && maxKern == 0 {
		fmt.Printf("Kerning     : None\n")
	} else if kernEvalCount < sfntFont.NumGlyphs() {
		pairsRateStr := minFloatFmt2(float64(kernPairs)/10000)
		fmt.Printf("Kerning     : %s to %s units (%s%% of the first 1k pairs)\n", minFixedFmt2(minKern), minFixedFmt2(maxKern), pairsRateStr)
	} else {
		pairsRateStr := minFloatFmt2(float64(kernPairs)*100/float64(sfntFont.NumGlyphs()*sfntFont.NumGlyphs()))
		fmt.Printf("Kerning     : %s to %s units (%s%% of the pairs)\n", minFixedFmt2(minKern), minFixedFmt2(maxKern), pairsRateStr)
	}

	contours, err := sfntFont.LoadGlyph(&buffer, 0, unitSize, nil)
	if err != nil { log.Fatal(err) }
	if contours.Bounds().Empty() {
		fmt.Print("\n.notdef Glyph   : Empty\n")
	} else {
		fmt.Print("\n.notdef Glyph   : Non-Empty\n")
	}

	coloredGlyphs := make([]uint16, 0, 8)
	missingGlyphs := make([]uint16, 0, 8)
	var badLeft, badRight, badTop, badBottom fixed.Int26_6
	var badLeftIndex, badRightIndex, badTopIndex, badBottomIndex uint16
	numGlyphs := uint16(sfntFont.NumGlyphs())
	for i := uint16(0); i < numGlyphs; i++ {
		contours, err = sfntFont.LoadGlyph(&buffer, sfnt.GlyphIndex(i), unitSize, nil)
		if err != nil {
			if err == sfnt.ErrColoredGlyph {
				coloredGlyphs = append(coloredGlyphs, i)
			} else if err == sfnt.ErrNotFound {
				missingGlyphs = append(missingGlyphs, i)
			} else {
				log.Fatal(err)
			}
		} else {
			left, right, top, bottom := CBoxBadness(contours)
			if left   > badLeft   { badLeft   = left   ; badLeftIndex   = i }
			if right  > badRight  { badRight  = right  ; badRightIndex  = i }
			if top    > badTop    { badTop    = top    ; badTopIndex    = i }
			if bottom > badBottom { badBottom = bottom ; badBottomIndex = i }
		}
	}

	if len(coloredGlyphs) > 0 {
		fmt.Printf("Colored Glyphs  : %s\n", glyphIndexListFmt(coloredGlyphs))
	} else {
		fmt.Printf("Colored Glyphs  : No\n")
	}

	if len(missingGlyphs) > 0 {
		fmt.Printf("Missing Glyphs  : %s\n", glyphIndexListFmt(missingGlyphs))
	}

	if badLeft > 0 || badRight > 0 || badTop > 0 || badBottom > 0 {
		fmt.Printf("CtrlBox Badness :\n")
		if badLeft > 0 {
			fmt.Printf("   Left   : %d units (glyph %d)\n", badLeft, badLeftIndex)
		}
		if badRight > 0 {
			fmt.Printf("   Right  : %d units (glyph %d)\n", badRight, badRightIndex)
		}
		if badTop > 0 {
			fmt.Printf("   Top    : %d units (glyph %d)\n", badTop, badTopIndex)
		}
		if badBottom > 0 {
			fmt.Printf("   Bottom : %d units (glyph %d)\n", badBottom, badBottomIndex)
		}
	} else {
		fmt.Printf("CtrlBox Badness : Zero\n")
	}
}

func minFloatFmt2(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

func minFixedFmt2(f fixed.Int26_6) string {
	return minFloatFmt2(float64(f)/64.0)
}

func glyphIndexListFmt(indices []uint16) string {
	var strBuilder strings.Builder
	for i, index := range indices {
		if i > 0 { strBuilder.WriteString(", ") }
		strBuilder.WriteString(strconv.Itoa(int(index)))
	}
	return strBuilder.String()
}

// ---- property functions ----
// This existed inside the 'emetric' subpackage in previous versions of etxt,
// but it was so niche that I decided to remove it in later versions. Now it's
// a mini-library living within this example.

// Returns the ascent of the given rune both as units and as the ratio
// to the font's em square size. In general, capital latin latters will
// return ratios around 0.7, while lowercase letters like 'a', 'x', 'r'
// and similar will return ratios around 0.48. But anything is possible,
// really.
//
// The buffer can be nil.
func RuneAscent(sfntFont *sfnt.Font, codePoint rune, buffer *sfnt.Buffer) (sfnt.Units, float64, error) {
	if buffer == nil { buffer = &sfnt.Buffer{} }
	unitSize := fixed.Int26_6(sfntFont.UnitsPerEm())
	glyphIndex, err := sfntFont.GlyphIndex(buffer, codePoint)
	if err != nil { return 0, 0, err }
	contours, err := sfntFont.LoadGlyph(buffer, glyphIndex, unitSize, nil)
	if err != nil { return 0, 0, err }
	ascentUnits  := -contours.Bounds().Min.Y
	emProportion := float64(ascentUnits)/float64(unitSize)
	return sfnt.Units(ascentUnits), emProportion, nil
}

// Computes how much the [control box] of the given segments exceeds
// the box defined by the "ON" contour points. Whenever there's an
// excess, that means that the control box doesn't match the bounding
// box of the glyph segments, which might have unintended effects in
// the rendering position of the glyph. Though you'd have to be crazy
// to care much about this, as the effect is almost always way smaller
// than typical hinting distortions. So, visually you are unlikely to
// see anything at all even if CBoxBadness are non-zero... but it has
// implications for technical correctness of computed left and right
// side bearings and stuff like that if you are obsessive enough.
//
// Returned badnesses are left, right, top and bottom, and the values
// can only be zero or positive.
//
// [control box]: https://freetype.org/freetype2/docs/glyphs/glyphs-6.html#section-2
func CBoxBadness(segments sfnt.Segments) (fixed.Int26_6, fixed.Int26_6, fixed.Int26_6, fixed.Int26_6) {
	cbox, obox := cboxObox(segments)
	leftBadness   := -cbox.Min.X + cbox.Min.X
	rightBadness  :=  cbox.Max.X - obox.Max.X
	topBadness    := -cbox.Min.Y + obox.Min.Y
	bottomBadness :=  cbox.Max.Y - obox.Max.Y
	return leftBadness, rightBadness, topBadness, bottomBadness
}

// cboxObox computes two bounding boxes for the given segments:
//  - The [control box], equivalent to sfnt.Segments.Bounds().
//  - The "ON" contour points bounding box.
// These two can be used by CBoxBadness to determine if the CBox
// matches the real bounding box or not (though the actual bounding
// box can't be easily determined if the two are different).
//
// [control box]: https://freetype.org/freetype2/docs/glyphs/glyphs-6.html#section-2
func cboxObox(segments sfnt.Segments) (fixed.Rectangle26_6, fixed.Rectangle26_6) {
	// create boxes
	cbox := fixed.Rectangle26_6 {
		Min: fixed.Point26_6 {
			X: fixed.Int26_6(0x7FFFFFFF),
			Y: fixed.Int26_6(0x7FFFFFFF),
		},
		Max: fixed.Point26_6 {
			X: fixed.Int26_6(-0x80000000),
			Y: fixed.Int26_6(-0x80000000),
		},
	}
	obox := fixed.Rectangle26_6 {
		Min: fixed.Point26_6 { X: cbox.Min.X, Y: cbox.Min.Y },
		Max: fixed.Point26_6 { X: cbox.Max.X, Y: cbox.Max.Y },
	}

	// iterate segments
	for _, segment := range segments {
		switch segment.Op {
		case sfnt.SegmentOpMoveTo, sfnt.SegmentOpLineTo:
			adjustBoxLimits(&cbox, segment.Args[0 : 1])
			adjustBoxLimits(&obox, segment.Args[0 : 1])
		case sfnt.SegmentOpQuadTo:
			adjustBoxLimits(&cbox, segment.Args[0 : 2])
			adjustBoxLimits(&obox, segment.Args[1 : 2])
		case sfnt.SegmentOpCubeTo:
			adjustBoxLimits(&cbox, segment.Args[0 : 3])
			adjustBoxLimits(&obox, segment.Args[2 : 3])
		default:
			panic("unexpected segment.Op")
		}
	}
	return cbox, obox
}

func adjustBoxLimits(box *fixed.Rectangle26_6, points []fixed.Point26_6) {
	for _, point := range points {
		if box.Max.X < point.X { box.Max.X = point.X }
		if box.Min.X > point.X { box.Min.X = point.X }
		if box.Max.Y < point.Y { box.Max.Y = point.Y }
		if box.Min.Y > point.Y { box.Min.Y = point.Y }
	}
}
