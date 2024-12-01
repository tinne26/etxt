package etxt

import "testing"

func testFailRunes(t *testing.T, expected rune, got rune) {
	t.Fatalf("expected '%s', got '%s'", string(expected), string(got))
}

func TestRtlStringIterator(t *testing.T) {
	var iter rtlStringIterator
	testString := "abcd"
	iter.Init(testString)
	for _, expected := range []rune{'d', 'c', 'b', 'a', -1, -1, -1} {
		got := iter.Next(testString)
		if got != expected {
			testFailRunes(t, expected, got)
		}
	}

	testString = "a"
	iter.Init(testString)
	for _, expected := range []rune{'a', -1, -1, -1} {
		got := iter.Next(testString)
		if got != expected {
			testFailRunes(t, expected, got)
		}
	}

	testString = ""
	iter.Init(testString)
	for _, expected := range []rune{-1, -1, -1} {
		got := iter.Next(testString)
		if got != expected {
			testFailRunes(t, expected, got)
		}
	}

	testString = "\n"
	iter.Init(testString)
	for _, expected := range []rune{'\n', -1, -1, -1} {
		got := iter.Next(testString)
		if got != expected {
			testFailRunes(t, expected, got)
		}
	}

	testString = "\n\n"
	iter.Init(testString)
	for _, expected := range []rune{'\n', '\n', -1, -1, -1} {
		got := iter.Next(testString)
		if got != expected {
			testFailRunes(t, expected, got)
		}
	}

	testString = "\na\nb\n"
	iter.Init(testString)
	for _, expected := range []rune{'\n', 'a', '\n', 'b', '\n', -1, -1, -1} {
		got := iter.Next(testString)
		if got != expected {
			testFailRunes(t, expected, got)
		}
	}

	testString = "hello\nworld\n"
	iter.Init(testString)
	for _, expected := range []rune{'o', 'l', 'l', 'e', 'h', '\n', 'd', 'l', 'r', 'o', 'w', '\n', -1, -1, -1} {
		got := iter.Next(testString)
		if got != expected {
			testFailRunes(t, expected, got)
		}
	}
}
