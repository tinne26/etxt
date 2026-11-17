package font

// This file gives tests access to the fonts placed in test/, picked by the
// glyphs each test needs, and provides some helper methods.

import (
	"embed"
	"os"
	"testing"

	"github.com/tinne26/etxt/internal/testfont"
)

//go:embed test/*
var testfs embed.FS

const testFontsDir = "test"

var testFonts = testfont.NewDir(testfs, testFontsDir)

func TestMain(m *testing.M) {
	os.Exit(testFonts.ReportSkips(m.Run()))
}

func doesNotPanic(function func()) (didNotPanic bool) {
	didNotPanic = true
	defer func() { didNotPanic = (recover() == nil) }()
	function()
	return
}
