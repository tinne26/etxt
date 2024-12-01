package font

import "os"
import "io"
import "io/fs"
import "errors"

import "golang.org/x/image/font/sfnt"

// Similar to [sfnt.Parse](), but also including the font name
// in the returned values. The bytes must not be modified while
// the font is in use.
//
// This is a low level function; you may prefer to use a
// [Library] instead.
//
// [sfnt.Parse]: https://pkg.go.dev/golang.org/x/image/font/sfnt#Parse.
func ParseFromBytes(fontBytes []byte) (*sfnt.Font, string, error) {
	newFont, err := sfnt.Parse(fontBytes)
	if err != nil {
		return nil, "", err
	}
	fontName, err := GetName(newFont)
	return newFont, fontName, err
}

// Attempts to parse a font located the given filepath and returns it
// along its name and any possible error. Supported formats are .ttf
// and .otf.
//
// This is a low level function; you may prefer to use a
// [Library] instead.
func ParseFromPath(path string) (*sfnt.Font, string, error) {
	// check font path validity
	ok := hasValidFontExtension(path)
	if !ok {
		return nil, "", errors.New("invalid font path '" + path + "'")
	}

	// open font file
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	return parseFontFileAndClose(file)
}

// Same as [ParseFromPath](), but for embedded filesystems.
//
// This is a low level function; you may prefer to use a
// [Library] instead.
func ParseFromFS(filesys fs.FS, path string) (*sfnt.Font, string, error) {
	// check font path validity
	ok := hasValidFontExtension(path)
	if !ok {
		return nil, "", errors.New("invalid font path '" + path + "'")
	}

	// open font file
	file, err := filesys.Open(path)
	if err != nil {
		return nil, "", err
	}
	return parseFontFileAndClose(file)
}

// ---- helpers ----

func parseFontFileAndClose(file io.ReadCloser) (*sfnt.Font, string, error) {
	fontBytes, err := io.ReadAll(file)
	if err != nil {
		_ = file.Close()
		return nil, "", err
	}
	err = file.Close()
	if err != nil {
		return nil, "", err
	}
	return ParseFromBytes(fontBytes)
}

// Whether font path ends in .ttf or .otf.
func hasValidFontExtension(path string) bool {
	if len(path) < 4 {
		return false
	}
	if path[len(path)-1] != 'f' {
		return false
	}
	if path[len(path)-2] != 't' {
		return false
	}
	thrd := path[len(path)-3]
	if thrd != 't' && thrd != 'o' {
		return false
	}
	if path[len(path)-4] != '.' {
		return false
	}
	return true
}
