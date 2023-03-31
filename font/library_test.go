package font

import "io"
import "errors"
import "testing"
import "golang.org/x/image/font/sfnt"

func TestLibrary(t *testing.T) {
	lib := NewLibrary()
	if lib.Size() != 0 { t.Fatal("really?") }

	ensureTestAssetsLoaded()
	if testFontA == nil { t.SkipNow() }

	added, skipped, err := lib.ParseAllFromPath(testFontsDir + "/" + testPathA)
	if err != nil { t.Fatalf("unexpected error: %s", err.Error()) }
	if added   != 1 { t.Fatal("expected 1 added font") }
	if skipped != 0 { t.Fatal("expected 0 skipped fonts") }

	font, name, err := ParseFromPath(testFontsDir + "/" + testPathA)
	if !lib.HasFont(name) {
		t.Fatalf("expected Library to include %s", name)
	}

	if lib.GetFont(name) == nil {
		t.Fatal("expected Library to allow access to the font")
	}

	if lib.GetFont("SurelyYouDontNameYourFontsLikeThis_") != nil {
		t.Fatal("well, well, well...")
	}

	lib.EachFont(func(fname string, _ *sfnt.Font) error {
		if fname != name { t.Fatalf("unexpected font %s", fname) }
		return nil
	})
	if lib.RemoveFont("totally-not-fake-yay") { t.Fatal("unexpected remove") }
	if !lib.RemoveFont(name) { t.Fatal("unexpected remove failure") }
	lib.EachFont(func(fname string, _ *sfnt.Font) error {
		t.Fatalf("unexpected font %s", fname)
		return nil
	})

	_, err = lib.ParseFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	if err == nil { t.Fatal("expected error to be non-nil") }

	added, skipped, err = lib.ParseAllFromFS(testfs, testFontsDir)
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
		t.Logf("WARNING: skipped %d fonts during embed parsing. Do you have dup fonts on %s?", skipped, testFontsDir)
	}

	fname, err := lib.ParseFromFS(testfs, testFontsDir + "/" + testPathA)
	if err != ErrAlreadyPresent {
		t.Fatalf("expected ErrAlreadyPresent, got '%s'", err.Error())
	}
	if fname != name {
		t.Fatalf("expected '%s', got '%s'", name, fname)
	}

	if !lib.RemoveFont(name) {
		t.Fatalf("expected font %s to be present and possible to remove", name)
	}
	file, err := testfs.Open(testFontsDir + "/" + testPathA)
	if err != nil { file.Close() ; panic(err) }
	bytes, err := io.ReadAll(file)
	file.Close()
	if err != nil { panic(err) }
	fname, err = lib.ParseFromBytes(bytes)
	if err != nil { t.Fatalf("unexpected error '%s'", err) }
	if fname != name { t.Fatalf("unexpected name '%s' (expected '%s')", fname, name) }
	lib.RemoveFont(fname)

	lname, err := lib.AddFont(font)
	if err != nil {
		t.Fatalf("unexpected error on AddFont(): %s", err.Error())
	}
	if lname != name {
		t.Fatalf("expected AddFont() name return to be '%s', but got '%s' instead", name, lname)
	}

	mustErr := true
	err = lib.EachFont(func(string, *sfnt.Font) error {
		if mustErr {
			mustErr = false
			return errors.New("manual error test")
		}
		t.Fatal("EachFont failed to stop on the first iteration")
		return nil
	})
	if err == nil || err.Error() != "manual error test" {
		t.Fatalf("expected \"manual error test\" error, but got \"%s\" instead", err)
	}

	if doesNotPanic(func() { lib.AddFont(nil) }) {
		t.Fatalf("lib.AddFont(nil) should have panicked")
	}
	releaseSfntBuffer(sfntBuffer) // critical cleanup after the panic

	added, skipped, err = lib.ParseAllFromPath("unexistent/path/ffs/dont-tell-me")
	if added != 0 { t.Fatalf("added != 0") }
	if skipped != 0 { t.Fatalf("skipped != 0") }
	if err == nil { t.Fatalf("seriously?") }
}
