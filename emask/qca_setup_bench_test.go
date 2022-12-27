//go:build bench

package emask

// This file contains a fake test ensuring that test assets are available.

import "os"
import "log"
import "io/fs"
import "errors"
import "strings"
import "path/filepath"
import "testing"
import "golang.org/x/image/font/sfnt"

var testWarnings string
func TestCompleteness(t *testing.T) {
	if len(testWarnings) > 0 {
		t.Fatalf("missing test assets\n%s", testWarnings)
	}
}

var benchFont *sfnt.Font
func init() { // parse benchmark fonts
	_, err := os.Stat("../test/fonts/")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Fatal(err)
		}
		_, err = os.Stat("test/fonts/")
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				log.Fatal(err)
			}
		} else {
			tryReadBenchFontFromDir("test/fonts/")
			if benchFont != nil { return } // done
		}
	} else { // try read font from /test/fonts/
		tryReadBenchFontFromDir("../test/fonts/")
		if benchFont != nil { return } // done
	}

	// couldn't find .ttf yet, try work dir search
	tryReadBenchFontFromDir(".")
	if benchFont != nil { return } // done

	// failed to find font
	testWarnings = "WARNING: Expected a .ttf font in test/fonts/ or emask/\n" +
	               "WARNING: Benchmarks will be skipped\n"
}

func tryReadBenchFontFromDir(dir string) {
	absDir, err := filepath.Abs(dir)
	if err != nil { log.Fatal(err) }
	err = filepath.WalkDir(absDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil { return err }
		if benchFont != nil { return nil }
		if entry.IsDir() && path != absDir { return fs.SkipDir }
		if strings.HasSuffix(entry.Name(), ".ttf") {
			fontBytes, err := os.ReadFile(absDir + string(os.PathSeparator) + entry.Name())
			if err != nil { log.Fatal(err) }
			benchFont, err = sfnt.Parse(fontBytes)
			if err != nil { log.Fatal(err) }
		}
		return nil
	})
	if err != nil { log.Fatal(err) }
}
