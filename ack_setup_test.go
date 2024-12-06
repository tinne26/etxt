package etxt

// This file contains a fake test ensuring that test assets are available,
// setups a few important variables and provides some helper methods.

import (
	"embed"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"

	"github.com/tinne26/etxt/font"
	"golang.org/x/image/font/sfnt"
)

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
	if testAssetsLoaded {
		return
	}
	testAssetsLoaded = true

	// load library from embedded folder and check fonts
	lib := font.NewLibrary()
	_, _, err := lib.ParseAllFromFS(testfs, testFontsDir)
	if err != nil {
		fmt.Printf("TESTS INIT: %s", err.Error())
		os.Exit(1)
	}

	type FontInfo struct {
		font *sfnt.Font
		name string
	}
	fonts := make([]FontInfo, 0, 2)
	lib.EachFont(func(name string, sfntFont *sfnt.Font) error {
		fonts = append(fonts, FontInfo{sfntFont, name})
		return nil
	})
	sort.Slice(fonts, func(i, j int) bool {
		return fonts[i].name < fonts[j].name
	})

	// set fonts and/or warnings for missing fonts
	switch len(fonts) {
	case 0:
		testWarnings = "WARNING: Expected at least 2 .ttf fonts in " + testFontsDir + "/ (found 0)\n" +
			"WARNING: Most tests will be skipped\n"
	case 1:
		testFontA = fonts[0].font
		testWarnings = "WARNING: Expected at least 2 .ttf fonts in " + testFontsDir + "/ (found 1)\n" +
			"WARNING: Some tests will be skipped\n"
	default:
		testFontA = fonts[0].font
		testFontB = fonts[1].font
	}
}
