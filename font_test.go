package etxt

import "os"
import "embed"
import "io/fs"
import "errors"
import "strings"
import "log"

import "testing"

func init() {
	_, err := os.Stat("test_font.ttf")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Fatal("etxt requires a test_font.ttf file to run tests")
		}
		log.Fatal(err)
	}
}

//go:embed test_font.ttf
var embedFilesys embed.FS

func TestFontLibrary(t *testing.T) {
	lib := NewFontLibrary()
	if lib.Size() != 0 { t.Fatal("really?") }
	added, skipped, err := lib.ParseDirFonts("test_font.ttf")
	if err != nil { panic(err) }
	if added   != 1 { t.Fatal("expected 1 added font") }
	if skipped != 0 { t.Fatal("expected 0 skipped fonts") }

	font, name, err := ParseFontFrom("test_font.ttf")
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

	added, skipped, err = lib.ParseEmbedDirFonts(".", &embedFilesys)
	if err != nil { panic(err) }
	if added   != 1 { t.Fatal("expected 1 added font") }
	if skipped != 0 { t.Fatal("expected 0 skipped fonts") }

	fname, err := lib.ParseEmbedFontFrom("test_font.ttf", &embedFilesys)
	if err != ErrAlreadyLoaded {
		t.Fatalf("expected ErrAlreadyLoaded, got '%s'", err.Error())
	}
	if fname != name {
		t.Fatalf("expected '%s', got '%s'", name, fname)
	}
}

func TestGzip(t *testing.T) {
	const TestDirName = "test_gzip_fonts"

	// prepare mock directory and file
	err := os.Mkdir(TestDirName, 0777)
	if err != nil { panic(err) }

	file, err := os.Create(TestDirName + "/font.ttf")
	if err != nil { panic(err) }

	bytes, err := os.ReadFile("test_font.ttf")
	if err != nil { panic(err) }

	_, err = file.Write(bytes)
	if err != nil { panic(err) }
	err = file.Close()
	if err != nil { panic(err) }

	// defer cleanup
	defer func() {
		err := os.Remove(TestDirName + "/font.ttf")
		if err != nil { panic(err) }
		err  = os.Remove(TestDirName + "/font.ttf.gzip")
		if err != nil { panic(err) }
		err  = os.Remove(TestDirName)
		if err != nil { panic(err) }
	}()

	// test gzip dir fonts
	err = GzipDirFonts(TestDirName, TestDirName)
	if err != nil { t.Fatalf("GzipDirFonts failed: %s", err.Error()) }

	_, err = os.Stat(TestDirName + "/font.ttf.gzip")
	if err != nil {
		t.Fatalf("Checking the gzipped font failed: %s", err.Error())
	}

	_, err = os.Stat(TestDirName + "/font.ttf")
	if err != nil {
		t.Fatalf("Checking the original font failed: %s", err.Error())
	}

	_, nameTTF, err := ParseFontFrom(TestDirName + "/font.ttf")
	if err != nil {
		t.Fatalf("ParseFontFrom error for font: %s", err.Error())
	}

	_, nameGzip, err := ParseFontFrom(TestDirName + "/font.ttf.gzip")
	if err != nil {
		t.Fatalf("ParseFontFrom error for gzipped font: %s", err.Error())
	}

	if nameTTF != nameGzip {
		t.Fatalf("expected nameTTF == nameGzip (%s == %s)", nameTTF, nameGzip)
	}
}
