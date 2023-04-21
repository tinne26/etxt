package etxt

// This file contains a fake test ensuring that test assets are available,
// setups a few important variables and provides some helper methods.

import "os"
import "fmt"
import "embed"
import "sync"
import "testing"

import "github.com/tinne26/etxt/font"
import "golang.org/x/image/font/sfnt"

//go:embed font/test/*
var testfs embed.FS

var testFontsDir string = "font/test"
var testFontA *sfnt.Font
var testFontB *sfnt.Font
var assetsLoadMutex sync.Mutex
var testAssetsLoaded bool

func TestAssetAvailability(t *testing.T) {
	ensureTestAssetsLoaded()
	if len(testWarnings) > 0 {
		t.Fatalf("missing test assets\n%s", testWarnings)
	}
}

var testWarnings string
func ensureTestAssetsLoaded() {
	// assets load access control
	assetsLoadMutex.Lock()
	defer assetsLoadMutex.Unlock()
	if testAssetsLoaded { return }
	testAssetsLoaded = true

	// load library from embedded folder and check fonts
	lib := font.NewLibrary()
	_, _, err := lib.ParseAllFromFS(testfs, testFontsDir)
	if err != nil {
		fmt.Printf("TESTS INIT: %s", err.Error())
		os.Exit(1)
	}

	lib.EachFont(func(name string, sfntFont *sfnt.Font) error {
		if testFontA == nil {
			testFontA = sfntFont
			return nil
		} else {
			testFontB = sfntFont
			return font.ErrBreakEach
		}
	})

	// test missing data warnings
	if testFontA == nil {
		testWarnings = "WARNING: Expected at least 2 .ttf fonts in " + testFontsDir + "/ (found 0)\n" +
		               "WARNING: Most tests will be skipped\n"
	} else if testFontB == nil {
		testWarnings = "WARNING: Expected at least 2 .ttf fonts in " + testFontsDir + "/ (found 1)\n" +
		               "WARNING: Some tests will be skipped\n"
	}
}
