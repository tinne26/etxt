//go:build gtxt

package main

import "os"
import "log"
import "fmt"
import "strconv"
import "strings"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/emetric"

// Must be compiled with '-tags gtxt'

// This program prints info about the given font directly
// to standard output. Mostly metrics.

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	font, fontName, err := etxt.ParseFontFrom(os.Args[1])
	if err != nil { log.Fatal(err) }

	// start collecting and printinf font info
	var buffer sfnt.Buffer
	fmt.Printf("# %s\n", fontName)

	info, err := etxt.FontIdentifier(font)
	if err != nil {
		if err != etxt.ErrNotFound { log.Fatal(err) }
		info = "(not found)"
	}
	fmt.Printf("Identifier  : %s\n", info)
	fmt.Printf("Num. Glyphs : %d\n\n", font.NumGlyphs())

	info, err = etxt.FontFamily(font)
	if err != nil {
		if err != etxt.ErrNotFound { log.Fatal(err) }
		info = "(not found)"
	}
	fmt.Printf("Family  : %s\n", info)

	info, err = etxt.FontSubfamily(font)
	if err != nil {
		if err != etxt.ErrNotFound { log.Fatal(err) }
		info = "(not found)"
	}
	fmt.Printf("Style   : %s\n", info)

	postTable := font.PostTable()
	if postTable.ItalicAngle != 0 {
		fmt.Printf("Slant   : %s degrees\n", minFloatFmt2(postTable.ItalicAngle))
	}
	if postTable.IsFixedPitch {
		fmt.Print("Spacing : Monospaced\n")
	} else {
		fmt.Print("Spacing : Proportional\n")
	}
	fmt.Printf("\nEm Square   : %d units\n", font.UnitsPerEm())

	const NoHinting = 0
	unitSize := fixed.Int26_6(font.UnitsPerEm())
	metrics, err := font.Metrics(&buffer, unitSize, NoHinting)
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font Height : %4d units\n", metrics.Height)
	fmt.Printf("Ascent      : %4d units\n", metrics.Ascent)
	fmt.Printf("Descent     : %4d units\n", metrics.Descent)
	fmt.Printf("Line Gap    : %4d units\n", metrics.Height - metrics.Ascent - metrics.Descent)
	fmt.Printf("Cap. Height : %4d units (%s%% em height)\n", metrics.CapHeight, minFloatFmt2(100*float64(metrics.CapHeight)/float64(unitSize)))
	fmt.Printf("XHeight     : %4d units (%s%% em height)\n", metrics.XHeight, minFloatFmt2(100*float64(metrics.XHeight)/float64(unitSize)))
	ascUn, ascent, err := emetric.RuneAscent(font, 'T', &buffer)
	if err == nil { fmt.Printf("   Actual 'T' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = emetric.RuneAscent(font, 'A', &buffer)
	if err == nil { fmt.Printf("   Actual 'A' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = emetric.RuneAscent(font, 'O', &buffer)
	if err == nil { fmt.Printf("   Actual 'O' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = emetric.RuneAscent(font, 'x', &buffer)
	if err == nil { fmt.Printf("   Actual 'x' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = emetric.RuneAscent(font, 'a', &buffer)
	if err == nil { fmt.Printf("   Actual 'a' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }
	ascUn, ascent, err  = emetric.RuneAscent(font, 'r', &buffer)
	if err == nil { fmt.Printf("   Actual 'r' Height : %d units (%s%% em height)\n", ascUn, minFloatFmt2(100*ascent)) }

	minKern, maxKern := fixed.Int26_6(0), fixed.Int26_6(0)
	kernPairs := 0
	kernEvalCount := font.NumGlyphs()
	if kernEvalCount > 1000 { kernEvalCount = 1000 }
	for i := 0; i < kernEvalCount; i++ {
		for j := 0; j < kernEvalCount; j++ {
			kern, err := font.Kern(&buffer, etxt.GlyphIndex(i), etxt.GlyphIndex(j), unitSize, NoHinting)
			if err != nil {
				if err == sfnt.ErrNotFound { continue }
				log.Fatal(err)
			}
			if kern < minKern { minKern = kern }
			if kern > maxKern { maxKern = kern }
			kernPairs += 1
		}
	}
	if minKern == 0 && maxKern == 0 {
		fmt.Printf("Kerning     : None\n")
	} else if kernEvalCount < font.NumGlyphs() {
		pairsRateStr := minFloatFmt2(float64(kernPairs)/10000)
		fmt.Printf("Kerning     : %s to %s units (%s%% of the first 1k pairs)\n", minFloatFmt2(float64(minKern)/64), minFloatFmt2(float64(maxKern)/64), pairsRateStr)
	} else {
		pairsRateStr := minFloatFmt2(float64(kernPairs)*100/float64(font.NumGlyphs()*font.NumGlyphs()))
		fmt.Printf("Kerning     : %s to %s units (%s%% of the pairs)\n", minFloatFmt2(float64(minKern)/64), minFloatFmt2(float64(maxKern)/64), pairsRateStr)
	}

	contours, err := font.LoadGlyph(&buffer, 0, unitSize, nil)
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
	numGlyphs := uint16(font.NumGlyphs())
	for i := uint16(0); i < numGlyphs; i++ {
		contours, err = font.LoadGlyph(&buffer, sfnt.GlyphIndex(i), unitSize, nil)
		if err != nil {
			if err == sfnt.ErrColoredGlyph {
				coloredGlyphs = append(coloredGlyphs, i)
			} else if err == sfnt.ErrNotFound {
				missingGlyphs = append(missingGlyphs, i)
			} else {
				log.Fatal(err)
			}
		} else {
			left, right, top, bottom := emetric.CBoxBadness(contours)
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

func glyphIndexListFmt(indices []uint16) string {
	var strBuilder strings.Builder
	for i, index := range indices {
		if i > 0 { strBuilder.WriteString(", ") }
		strBuilder.WriteString(strconv.Itoa(int(index)))
	}
	return strBuilder.String()
}
