package font

import (
	"strings"
	"testing"
)

func TestGetProperties(t *testing.T) {
	font := testFonts.WithRunes(t, "")[0].Font
	var value string
	var err error

	// ensure state sanity
	buffer := getSfntBuffer()
	if buffer == nil {
		panic("unexpected nil")
	}
	releaseSfntBuffer(buffer)

	// test unexsitent property
	value, err = GetProperty(font, 999)
	if err != ErrNotFound {
		t.Fatalf("GetProperty(font, 999) error: %s", err)
	}
	if value != "" {
		t.Fatalf("GetProperty(nil, 999) value = \"%s\"", value)
	}

	name, err := GetName(font)
	if err != nil {
		panic(err)
	}
	ident, err := GetIdentifier(font)
	if err != nil {
		panic(err)
	}
	family, err := GetFamily(font)
	if err != nil {
		panic(err)
	}
	if !strings.Contains(name, family) && !strings.Contains(ident, family) {
		holyBible := "expected font name (%s) or identifier (%s) to contain "
		holyBible += "font family (%s). Maybe you are using a weird font?"
		t.Fatalf(holyBible, name, ident, family)
	}
	subfamily, err := GetSubfamily(font)
	if err != nil {
		panic(err)
	}
	if subfamily != "Regular" && subfamily != "Italic" &&
		subfamily != "Bold" && subfamily != "Bold Italic" {
		t.Fatalf("expected a... normal font subfamily, but got %s", subfamily)
	}

	if sfntBuffer == nil {
		panic("unexpected nil")
	}
	buffer = getSfntBuffer()
	if buffer == nil {
		panic("failed to get shared sfntBuffer")
	}
	ident2, err := GetIdentifier(font)
	if err != nil {
		panic(err)
	}
	if ident2 != ident {
		t.Fatalf("ident2 != ident")
	}
	releaseSfntBuffer(buffer)
}

func TestGetMissingRunes(t *testing.T) {
	font := testFonts.WithRunes(t, " ")[0].Font
	var missing []rune
	var err error

	missing, err = GetMissingRunes(font, " ")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(missing) != 0 {
		t.Fatalf("unexpected missing runes: %v", missing)
	}
	missing, err = GetMissingRunes(font, "\uF800")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(missing) != 1 {
		t.Fatal("unexpected rune \"\\uF800\" not missing")
	}

	missing, err = GetMissingRunes(font, " \uF800 \uF800\uF800    ")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(missing) != 3 {
		t.Fatalf("unexpected len(missing) == %d", len(missing))
	}

	// NOTE: missing font.GlyphIndex(buffer, codePoint) test, but I can't
	//       come up with an easy way to test that (without creating a dummy
	//       font intended to trigger the error)
}
