//go:build test

package etxt

// This file contains a fake test ensuring that test assets are available,
// setups a few important variables and provides some helper methods.

import "os"
import "fmt"
import "log"
import "strings"
import "image"
import "image/png"
import "embed"
import "testing"

//go:embed test/fonts/*
var testfs embed.FS

var testPathA string
var testFontA *Font
var testFontB *Font

func TestCompleteness(t *testing.T) {
	if len(testWarnings) > 0 {
		t.Fatalf("missing test assets\n%s", testWarnings)
	}
}

var testWarnings string
func init() {
	var failInit = func(err error) {
		fmt.Printf("TESTS INIT: %s", err.Error())
		os.Exit(1)
	}

	// parse embedded directory and check for useful fonts
	entries, err := testfs.ReadDir("test/fonts")
	if err != nil { failInit(err) }

	// manual loading to avoid depending on font library here
	var nameA, nameB string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".ttf") {
			if testFontA == nil {
				testPathA = entry.Name()
				testFontA, nameA, err = ParseEmbedFontFrom("test/fonts/" + entry.Name(), testfs)
				if err != nil { failInit(err) }
			} else { // testFontB == nil
				testFontB, nameB, err = ParseEmbedFontFrom("test/fonts/" + entry.Name(), testfs)
				if err != nil { failInit(err) }
				if nameA != nameB { break } // stop loading fonts
				testFontB = nil
			}
		}
	}
	
	// test missing data warnings
	if testFontA == nil {
		testWarnings = "WARNING: Expected at least 2 .ttf fonts in test/fonts/ (found 0)\n" +
		               "WARNING: Most tests will be skipped\n"
	} else {
		if testFontB == nil {
			testWarnings = "WARNING: Expected at least 2 .ttf fonts in test/fonts/ (found 1)\n"
			if nameA == nameB {
				testWarnings += "WARNING: Found repeated font in test/fonts/\n"
			}
			testWarnings += "WARNING: Some tests will be skipped\n"
		}
	}
}

func doesNotPanic(function func()) (didNotPanic bool) {
	didNotPanic = true
	defer func() { didNotPanic = (recover() == nil) }()
	function()
	return
}

func debugExport(name string, img image.Image) {
	file, err := os.Create(name)
	if err != nil { log.Fatal(err) }
	err = png.Encode(file, img)
	if err != nil { log.Fatal(err) }
	err = file.Close()
	if err != nil { log.Fatal(err) }
}
