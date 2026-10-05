package etxt

import (
	"testing"
	"unicode"

	"golang.org/x/image/font/sfnt"
)

func TestScriptFontAssignment(t *testing.T) {
	if testFontA == nil || testFontB == nil {
		t.SkipNow()
	}

	renderer := NewRenderer()
	renderer.SetFont(testFontA)

	if renderer.Script().GetFont(unicode.Han) != nil {
		t.Fatal("expected no font for an unassigned script")
	}

	renderer.Script().SetFont(unicode.Han, testFontB)
	if renderer.Script().GetFont(unicode.Han) != testFontB {
		t.Fatal("expected testFontB for Han")
	}
	if renderer.GetFont() != testFontA {
		t.Fatal("assigning a script font must not change the active font")
	}

	// a script can be reassigned, and one font can serve many scripts
	renderer.Script().SetFont(unicode.Han, testFontA)
	renderer.Script().SetFont(unicode.Cyrillic, testFontA)
	if renderer.Script().GetFont(unicode.Han) != testFontA {
		t.Fatal("expected Han reassignment to testFontA")
	}
	if renderer.Script().GetFont(unicode.Cyrillic) != testFontA {
		t.Fatal("expected testFontA for Cyrillic")
	}

	// a nil font removes the assignment, and only that one
	renderer.Script().SetFont(unicode.Han, nil)
	if renderer.Script().GetFont(unicode.Han) != nil {
		t.Fatal("expected the Han assignment to be removed")
	}
	if renderer.Script().GetFont(unicode.Cyrillic) != testFontA {
		t.Fatal("expected the Cyrillic assignment to survive")
	}

	renderer.Script().Clear()
	if renderer.Script().GetFont(unicode.Cyrillic) != nil {
		t.Fatal("expected Clear() to remove every assignment")
	}
}

func TestScriptEach(t *testing.T) {
	if testFontA == nil || testFontB == nil {
		t.SkipNow()
	}

	renderer := NewRenderer()
	renderer.Script().SetFont(unicode.Han, testFontA)
	renderer.Script().SetFont(unicode.Greek, testFontB)
	renderer.Script().SetFont(unicode.Cyrillic, testFontA)

	var names []string
	var scripts []*unicode.RangeTable
	var fonts []*sfnt.Font
	renderer.Script().Each(func(name string, script *unicode.RangeTable, font *sfnt.Font) bool {
		names = append(names, name)
		scripts = append(scripts, script)
		fonts = append(fonts, font)
		return true
	})

	expectedNames := []string{"Cyrillic", "Greek", "Han"}
	expectedScripts := []*unicode.RangeTable{unicode.Cyrillic, unicode.Greek, unicode.Han}
	expectedFonts := []*sfnt.Font{testFontA, testFontB, testFontA}
	if len(names) != len(expectedNames) {
		t.Fatalf("expected %d assignments, got %d", len(expectedNames), len(names))
	}
	for i := range expectedNames {
		if names[i] != expectedNames[i] || scripts[i] != expectedScripts[i] || fonts[i] != expectedFonts[i] {
			t.Fatalf("unexpected assignment at index %d: %s", i, names[i])
		}
	}

	count := 0
	renderer.Script().Each(func(string, *unicode.RangeTable, *sfnt.Font) bool {
		count += 1
		return false
	})
	if count != 1 {
		t.Fatalf("expected Each() to stop on false, ran %d times", count)
	}
}

func TestScriptInvalidScriptPanics(t *testing.T) {
	if testFontA == nil {
		t.SkipNow()
	}

	invalid := map[string]*unicode.RangeTable{
		"nil":         nil,
		"Ideographic": unicode.Ideographic, // a property, not a script
	}
	for name, script := range invalid {
		if doesNotPanic(func() { NewRenderer().Script().SetFont(script, testFontA) }) {
			t.Fatalf("expected the %s script to panic", name)
		}
	}
}
