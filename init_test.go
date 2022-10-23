//go:build gtxt

package etxt

import "log"
import "io/fs"
import "errors"

var testFont *Font
var testFont2 *Font
func init() {
	var err error
	var fontName1, fontName2 string
	testFont, fontName1, err = ParseFontFrom("test_font.ttf")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) { log.Fatal(err) }
		log.Fatal("etxt requires a test_font.ttf file to run tests")
	}
	testFont2, fontName2, err = ParseFontFrom("test_font2.ttf")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) { log.Fatal(err) }
		log.Fatal("etxt requires a test_font2.ttf file to run tests")
	}

	if fontName1 == fontName2 {
		log.Fatal("etxt requires a test_font.ttf and test_font2.ttf to be different fonts")
	}
}

func doesNotPanic(function func()) (didNotPanic bool) {
	didNotPanic = true
	defer func() { didNotPanic = (recover() == nil) }()
	function()
	return
}
