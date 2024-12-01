package font

// This file contains a fake test ensuring that test assets are available,
// setups a few important variables and provides some helper methods.

import "os"
import "fmt"
import "embed"
import "sync"
import "testing"
import "golang.org/x/image/font/sfnt"

//go:embed test/*
var testfs embed.FS

var testFontsDir string = "test"
var testPathA string
var testFontA *sfnt.Font
var testFontB *sfnt.Font
var assetsLoadMutex sync.Mutex
var testAssetsLoaded bool

func TestCompleteness(t *testing.T) {
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

	// parse embedded directory and check for useful fonts
	entries, err := testfs.ReadDir(testFontsDir)
	if err != nil {
		fmt.Printf("TESTS INIT: %s", err)
		os.Exit(1)
	}

	// manual loading to avoid depending on font library here
	var mainFontName string
	for _, entry := range entries {
		entryName := entry.Name()
		if !hasValidFontExtension(entryName) {
			continue
		}
		path := testFontsDir + "/" + entryName
		font, fontName, err := ParseFromFS(testfs, path)
		if err != nil {
			fmt.Printf("TESTS INIT: %s", err)
			os.Exit(1)
		}

		if testFontA == nil {
			testFontA = font
			testPathA = entryName
			mainFontName = fontName
		} else {
			if mainFontName == fontName {
				continue
			}
			testFontB = font
			break
		}
	}

	// test missing data warnings
	if testFontA == nil {
		testWarnings = "WARNING: Expected at least 2 .ttf fonts in " + testFontsDir + "/ (found 0)\n" +
			"WARNING: Most tests will be skipped\n"
	} else if testFontB == nil {
		testWarnings = "WARNING: Expected at least 2 .ttf fonts in " + testFontsDir + "/ (found 1)\n" +
			"WARNING: Some tests will be skipped\n"
	}
}

func doesNotPanic(function func()) (didNotPanic bool) {
	didNotPanic = true
	defer func() { didNotPanic = (recover() == nil) }()
	function()
	return
}
