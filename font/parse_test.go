package font

import "io"
import "io/fs"
import "errors"
import "strings"
import "testing"

type fakeFS struct {}
func (fakeFS) Open(string) (fs.File, error) {
	return nil, errors.New("fakeFS")
}

type fakeReadCloser struct{ errOnRead bool }
func (self fakeReadCloser) Read(p []byte) (n int, err error) {
	if self.errOnRead { return 0, errors.New("fakeRead") }
	return 0, io.EOF
}
func (self fakeReadCloser) Close() error {
	return errors.New("fakeClose")
}

// Testing the tricky error cases, fundamentally. The main
// code paths are already implicitly tested through the library
// functions and tests.
func TestParse(t *testing.T) {
	var err error

	_, _, err = ParseFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	if err == nil { t.Fatal("expected error") }

	_, _, err = ParseFromPath("path/with/no/extension")
	if err == nil || !strings.Contains(err.Error(), "invalid font path") {
		t.Fatal("expected error with 'invalid font path' in its contents")
	}

	_, _, err = ParseFromPath("fake/path/must/not/exist/yay.ttf")
	if err == nil || !strings.Contains(err.Error(), "cannot find") {
		t.Fatal("expected error with 'cannot find' in its contents")
	}
	
	fakefs := fakeFS{}
	_, _, err = ParseFromFS(fakefs, "path/with/no/extension")
	if err == nil || !strings.Contains(err.Error(), "invalid font path") {
		t.Fatal("expected error with 'invalid font path' in its contents")
	}
	_, _, err = ParseFromFS(fakefs, "cool.ttf")
	if err == nil || err.Error() != "fakeFS" {
		t.Fatalf("expected \"fakeFS\" error, but got '%s'", err)
	}

	if hasValidFontExtension("") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension(".") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension(".t") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension(".tt") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension(".ttx") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension("ttf") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension("otf") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension(".tgf") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension(".gtf") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension(".mp4a") { t.Fatalf("not a valid font extension") }
	if hasValidFontExtension(".xttf") { t.Fatalf("not a valid font extension") }
	if !hasValidFontExtension(".ttf") { t.Fatalf(".ttf must be a valid font extension") }
	if !hasValidFontExtension(".otf") { t.Fatalf(".ttf must be a valid font extension") }

	rc := fakeReadCloser{ errOnRead: true }
	_, _, err = parseFontFileAndClose(rc)
	if err == nil || err.Error() != "fakeRead" {
		t.Fatalf("expected err == \"fakeRead\", but got '%s'", err)
	}
	rc.errOnRead = false
	_, _, err = parseFontFileAndClose(rc)
	if err == nil || err.Error() != "fakeClose" {
		t.Fatalf("expected err == \"fakeClose\", but got '%s'", err)
	}
}
