//go:build test

package etxt

import "os"
import "strings"
import "testing"

func TestFontLibrary(t *testing.T) {
	lib := NewFontLibrary()
	if lib.Size() != 0 { t.Fatal("really?") }
	if testFontA == nil { t.SkipNow() }

	added, skipped, err := lib.ParseDirFonts("test/fonts/" + testPathA)
	if err != nil { t.Fatalf("unexpected error: %s", err.Error()) }
	if added   != 1 { t.Fatal("expected 1 added font") }
	if skipped != 0 { t.Fatal("expected 0 skipped fonts") }

	font, name, err := ParseFontFrom("test/fonts/" + testPathA)
	if !lib.HasFont(name) {
		t.Fatalf("expected FontLibrary to include %s", name)
	}

	if lib.GetFont(name) == nil {
		t.Fatal("expected FontLibrary to allow access to the font")
	}

	if lib.GetFont("SurelyYouDontNameYourFontsLikeThis_") != nil {
		t.Fatal("well, well, well...")
	}

	ident, err  := FontIdentifier(font)
	if err != nil { panic(err) }
	family, err := FontFamily(font)
	if err != nil { panic(err) }
	if !strings.Contains(name, family) && !strings.Contains(ident, family) {
		holyBible := "expected font name (%s) or identifier (%s) to contain "
		holyBible += "font family (%s). Maybe you are using a weird font?"
		t.Fatalf(holyBible, name, ident, family)
	}

	subfamily, err := FontSubfamily(font)
	if err != nil { panic(err) }
	if subfamily != "Regular" && subfamily != "Italic" &&
	   subfamily != "Bold" && subfamily != "Bold Italic" {
			t.Fatalf("expected a... normal font subfamily, but got %s", subfamily)
	}

	lib.EachFont(func(fname string, _ *Font) error {
		if fname != name { t.Fatalf("unexpected font %s", fname) }
		return nil
	})
	lib.RemoveFont(name)
	lib.EachFont(func(fname string, _ *Font) error {
		t.Fatalf("unexpected font %s", fname)
		return nil
	})

	added, skipped, err = lib.ParseEmbedDirFonts("test/fonts", testfs)
	if err != nil { panic(err) }
	switch added {
	case 0: t.Fatal("expected at least 1 added font")
	case 1:
		if testFontB != nil {
			t.Fatal("expected at least 2 added fonts")
		}
	default:
		if testFontB == nil {
			t.Fatal("expected at most 1 added font, internal test init parsing mismatch")
			// ^ see init_test.go
		}
	}
	if skipped != 0 {
		t.Logf("WARNING: skipped %d fonts during embed parsing. Do you have dup fonts on test/fonts/?", skipped)
	}

	fname, err := lib.ParseEmbedFontFrom("test/fonts/" + testPathA, testfs)
	if err != ErrAlreadyLoaded {
		t.Fatalf("expected ErrAlreadyLoaded, got '%s'", err.Error())
	}
	if fname != name {
		t.Fatalf("expected '%s', got '%s'", name, fname)
	}

	if !lib.RemoveFont(name) {
		t.Fatalf("expected font %s to be present and possible to remove", name)
	}

	lname, err := lib.LoadFont(font)
	if err != nil {
		t.Fatalf("unexpected error on LoadFont(): %s", err.Error())
	}
	if lname != name {
		t.Fatalf("expected LoadFont() name return to be '%s', but got '%s' instead", name, lname)
	}

	if doesNotPanic(func() { lib.LoadFont(nil) }) {
		t.Fatalf("lib.LoadFont(nil) should have panicked")
	}
}

func TestGzip(t *testing.T) {
	if testFontA == nil { t.SkipNow() }

	// prepare directory and file
	dir := t.TempDir()
	file, err := os.Create(dir + "/font.ttf")
	if err != nil { panic(err) }

	bytes, err := os.ReadFile("test/fonts/" + testPathA)
	if err != nil { panic(err) }

	_, err = file.Write(bytes)
	if err != nil { panic(err) }
	err = file.Close()
	if err != nil { panic(err) }

	// test gzip dir fonts
	err = GzipDirFonts(dir, dir)
	if err != nil { t.Fatalf("GzipDirFonts failed: %s", err.Error()) }

	_, err = os.Stat(dir + "/font.ttf.gz")
	if err != nil {
		t.Fatalf("Checking the gzipped font failed: %s", err.Error())
	}

	_, err = os.Stat(dir + "/font.ttf")
	if err != nil {
		t.Fatalf("Checking the original font failed: %s", err.Error())
	}

	_, nameTTF, err := ParseFontFrom(dir + "/font.ttf")
	if err != nil {
		t.Fatalf("ParseFontFrom error for font: %s", err.Error())
	}

	_, nameGzip, err := ParseFontFrom(dir + "/font.ttf.gz")
	if err != nil {
		t.Fatalf("ParseFontFrom error for gzipped font: %s", err.Error())
	}

	if nameTTF != nameGzip {
		t.Fatalf("expected nameTTF == nameGzip (%s == %s) [ParseFontFrom]", nameTTF, nameGzip)
	}

	bytes, err = os.ReadFile(dir + "/font.ttf.gz")
	if err != nil { panic(err) }
	_, nameGzip, err = ParseFontBytes(bytes)
	if err != nil {
		t.Fatalf("ParseFontBytes error for gzipped font: %s", err.Error())
	}
	if nameTTF != nameGzip {
		t.Fatalf("expected nameTTF == nameGzip (%s == %s) [ParseFontBytes]", nameTTF, nameGzip)
	}
}
