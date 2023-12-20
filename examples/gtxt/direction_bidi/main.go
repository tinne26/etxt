//go:build gtxt

package main

import "os"
import "image"
import "image/color"
import "image/png"
import "path/filepath"
import "log"
import "fmt"
import "strings"

import "golang.org/x/text/unicode/bidi"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"

// Must be compiled with '-tags gtxt'

// Requires a font with both latin and arabic glyphs. For example, I
// used "El Messiri" (Mohamed Gaber / Jovanny Lemonad, really nice
// font!) to test, which should be available on google fonts if you
// want to try it.

// Please understand that this example is only a proof of concept, not
// a role model if you want to get bidi *right*. Among the limitations:
// - The arabic text is not being shaped.
// - The mirroring process is very simplified.
// - Kerning is not being applied between different ordering runs.
// - Some other subtle details are probably still wrong.
//
// You can create an html document with this content and open it in a
// browser if you want to see how it would look if it was done properly:
// <!DOCTYPE html><html><body><p dir="rtl" lang="ar">العاشر ليونيكود (Unicode Conference)، الذي سيعقد في 10-12 آذار 1997 مبدينة</p></body></html>

func main() {
	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// get the string to test ready
	// (example text taken from golang.org/x/text/unicode/bidi tests)
	bidiText := `العاشر ليونيكود (Unicode Conference)، الذي سيعقد في 10-12 آذار 1997 مبدينة`

	// verify that the font has both arabic and latin glyphs
	missingRunes, err := font.GetMissingRunes(sfntFont, bidiText)
	if err != nil { log.Fatal(err) }
	if len(missingRunes) != 0 {
		log.Print("This example requires a font with both latin and arabic glyphs.")
		log.Fatalf("Missing glyphs: %s", fmtMissingRunes(missingRunes))
	}


	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetSize(24)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.VertCenter | etxt.Left)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// determine right-to-left and left-to-right sections
	// (if you were more serious about bidi, you would roll
   // your own renderer wrapping the etxt renderer and have
   // all this encapsulated, but you get the idea...)
	bidiParagraph := bidi.Paragraph{}
	bidiParagraph.SetString(bidiText)
	ordering, err := bidiParagraph.Order()
	if err != nil { log.Fatal(err) }
	totalLength := 0
	for i := 0; i < ordering.NumRuns(); i++ {
		run := ordering.Run(i)
		str := run.String()
		dir := etxt.Direction(run.Direction())
		if dir == etxt.RightToLeft {
			str = applyMirroring(str)
		}
		renderer.SetDirection(dir)
		totalLength += renderer.Measure(str).IntWidth()
	}

	// create target image and fill it with white
	lineHeight := int(renderer.Utils().GetLineHeight()*1.4) // includes some padding
	sideMargin := 16 // margin for each side
	width := totalLength + sideMargin*2
	outImage := image.NewRGBA(image.Rect(0, 0, width, lineHeight))
	for i := 0; i < width*lineHeight*4; i++ { outImage.Pix[i] = 255 }

	// set target and prepare align and starting position
	origin := fract.IntsToPoint(0, lineHeight/2)
	if bidiParagraph.IsLeftToRight() {
		renderer.SetAlign(etxt.Left)
		origin.X = fract.FromInt(sideMargin)
	} else { // is right to left
		renderer.SetAlign(etxt.Right)
		origin.X = fract.FromInt(width - sideMargin)
	}

	// draw each ordering run
	for i := 0; i < ordering.NumRuns(); i++ {
		run := ordering.Run(i)
		dir := etxt.Direction(run.Direction())
		renderer.SetDirection(dir)

		str := run.String()
		if dir == etxt.RightToLeft {
			str = applyMirroring(str)
		}
		renderer.Fract().Draw(outImage, str, origin.X, origin.Y) // (missing kern!)
		origin.X -= renderer.Measure(str).Width()
	}

	// store image as png
	filename, err := filepath.Abs("gtxt_direction_bidi.png")
	if err != nil { log.Fatal(err) }
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil { log.Fatal(err) }
	err = png.Encode(file, outImage)
	if err != nil { log.Fatal(err) }
	err = file.Close()
	if err != nil { log.Fatal(err) }
	fmt.Print("Program exited successfully.\n")
}

// Characters like parentheses that belong to the neutral BIDI class
// and are mirrored have to be swapped with their mirrored counterparts
// if they appear in right-to-left text. There are many mirrored code
// points (see https://www.compart.com/en/unicode/mirrored), but this
// function only deals with a few common ones.
//
// The way this works is very inefficient, but the purpose of this
// function is only to showcase that this is a necessary step if you
// are trying to get bidirectional text right.
//
// Golang's bidi package also does mirroring in ReverseString(), but
// that reverses the whole string. We will be using the renderer's
// text direction for that instead.
func applyMirroring(str string) string {
	var strBuilder strings.Builder
	for _, codePoint := range str {
		switch codePoint {
		case '(': strBuilder.WriteRune(')')
		case ')': strBuilder.WriteRune('(')
		case '[': strBuilder.WriteRune(']')
		case ']': strBuilder.WriteRune('[')
		case '{': strBuilder.WriteRune('}')
		case '}': strBuilder.WriteRune('{')
		case '<': strBuilder.WriteRune('>')
		case '>': strBuilder.WriteRune('<')
		case '«': strBuilder.WriteRune('»')
		case '»': strBuilder.WriteRune('«')
		case '‹': strBuilder.WriteRune('›')
		case '›': strBuilder.WriteRune('‹')
		case '⟨': strBuilder.WriteRune('⟩')
		case '⟩': strBuilder.WriteRune('⟨')
		case '⟪': strBuilder.WriteRune('⟫')
		case '⟫': strBuilder.WriteRune('⟪')
		default:
			strBuilder.WriteRune(codePoint)
		}
	}
	return strBuilder.String()
}

// Remove GetMissingRunes() dups and format nicely.
func fmtMissingRunes(runes []rune) string {
	seen := make(map[rune]struct{})
	var strBuilder strings.Builder
	for i, codePoint := range runes {
		_, alreadySeen := seen[codePoint]
		if alreadySeen { continue }
		seen[codePoint] = struct{}{}
		if i > 0 { strBuilder.WriteString(", ") }
		strBuilder.WriteRune(codePoint)
	}
	return strBuilder.String()
}
