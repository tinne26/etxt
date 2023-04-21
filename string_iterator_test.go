package etxt

import "testing"

import "golang.org/x/image/font/sfnt"

func testFailRunes(t *testing.T, expected rune, got rune) {
	t.Fatalf("expected '%s', got '%s'", string(expected), string(got))
}

func testFailStr(t *testing.T, expected string, got string) {
	t.Fatalf("expected '%s', got '%s'", expected, got)
}

func TestStrIterator(t *testing.T) {
	// simple case
	iter := newStrIterator("one day", false)
	for _, expected := range []rune{'o', 'n', 'e', ' ', 'd', 'a', 'y', -1, -1, -1} {
		got := iter.Next()
		if got != expected { testFailRunes(t, expected, got) }
	}
	iter = newStrIterator("one day", false)
	expected, got := "one day", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }

	// line breaks
	iter = newStrIterator("0123\n0123\n", false)
	expected, got = "0123", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	for _, expected := range []rune{'0', '1', '2', '3'} {
		got := iter.Next()
		if got != expected { testFailRunes(t, expected, got) }
	}
	expected, got = "", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune := iter.Next()
	if gotRune != '\n' { testFailRunes(t, '\n', gotRune) }
	expected, got = "0123", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune  = iter.Next()
	if gotRune != '0' { testFailRunes(t, '0', gotRune) }
	expected, got = "123", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	for i := 0; i < 3; i++ { iter.Next() }
	expected, got = "", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune  = iter.Next()
	if gotRune != '\n' { testFailRunes(t, '\n', gotRune) }

	// no ending line break
	iter = newStrIterator("B\nA", false)
	expected, got = "B", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune = iter.Next()
	if gotRune != 'B' { testFailRunes(t, 'B', gotRune) }
	expected, got = "", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune = iter.Next()
	if gotRune != '\n' { testFailRunes(t, '\n', gotRune) }
	expected, got = "A", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune  = iter.Next()
	if gotRune != 'A' { testFailRunes(t, 'A', gotRune) }
	gotRune  = iter.Next()
	if gotRune != -1 { testFailRunes(t, -1, gotRune) }
	expected, got = "", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
}

func TestStrIteratorReverse(t *testing.T) {
	iter := newStrIterator("0123\nAB CD\n", true)
	expected, got := "0123", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	for _, expected := range []rune{'3', '2', '1', '0'} {
		got := iter.Next()
		if got != expected { testFailRunes(t, expected, got) }
	}
	expected, got = "", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune := iter.Next()
	if gotRune != '\n' { testFailRunes(t, '\n', gotRune) }
	expected, got = "AB CD", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune  = iter.Next()
	if gotRune != 'D' { testFailRunes(t, 'D', gotRune) }
	expected, got = "AB C", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	for _, expected := range []rune{'C', ' ', 'B', 'A'} {
		got := iter.Next()
		if got != expected { testFailRunes(t, expected, got) }
	}
	expected, got = "", iter.UntilNextLineBreak()
	if got != expected { testFailStr(t, expected, got) }
	gotRune  = iter.Next()
	if gotRune != '\n' { testFailRunes(t, '\n', gotRune) }
	gotRune  = iter.Next()
	if gotRune != -1 { testFailRunes(t, -1, gotRune) }
}

func TestGlyphsIterator(t *testing.T) {
	iter := newGlyphsIterator([]sfnt.GlyphIndex{1, 2, 3, 4}, false)
	for n, expected := range []sfnt.GlyphIndex{1, 2, 3, 4} {
		got, done := iter.Next()
		if done { t.Fatalf("test#%d unexpectedly done", n) }
		if got != expected {
			t.Fatalf("test#%d expected %d got %d", n, expected, got)
		}
	}

	got, done := iter.Next()
	if got != 0 || !done { t.Fatalf("expected done") }
}

func TestGlyphsIteratorReverse(t *testing.T) {
	iter := newGlyphsIterator([]sfnt.GlyphIndex{1, 2, 3, 4}, true)
	for n, expected := range []sfnt.GlyphIndex{4, 3, 2, 1} {
		got, done := iter.Next()
		if done { t.Fatalf("test#%d unexpectedly done", n) }
		if got != expected {
			t.Fatalf("test#%d expected %d got %d", n, expected, got)
		}
	}

	got, done := iter.Next()
	if got != 0 || !done { t.Fatalf("expected done") }
}
