package font

import "golang.org/x/image/font/sfnt"
import "sync/atomic"
import "errors"

var ErrNotFound = errors.New("font property not found or empty")

// We allocate one sfnt.Buffer so it can be used in FontProperty() calls.
// These buffers can't be used concurrently though, so sfntBuffer will only
// be used if no one else is using it at the moment. We don't bother creating
// a pool because that would be waaay overkill, and this simple trick already
// makes calls to FontProperty() fast in 99% use-cases for a reasonably small
// price. Only limitation? Panic recovers can leave it locked as being used.
var sfntBuffer *sfnt.Buffer
var usingSfntBuffer uint32 = 0
func getSfntBuffer() *sfnt.Buffer {
	if !atomic.CompareAndSwapUint32(&usingSfntBuffer, 0, 1) {
		return nil
	}
	if sfntBuffer == nil {
		sfntBuffer = &sfnt.Buffer{}
	}
	return sfntBuffer
}

func releaseSfntBuffer(buffer *sfnt.Buffer) {
	if buffer != nil {
		atomic.StoreUint32(&usingSfntBuffer, 0)
	}
}

// Returns the requested font property for the given font.
// The returned property string might be empty even when error is nil.
// If the property is missing, [ErrNotFound] will be returned.
func GetProperty(font *sfnt.Font, property sfnt.NameID) (string, error) {
	buffer := getSfntBuffer()
	str, err := font.Name(buffer, property)
	releaseSfntBuffer(buffer)
	if err == sfnt.ErrNotFound { return "", ErrNotFound }
	return str, err
}

// Returns the family name of the given font. If the information is
// missing, [ErrNotFound] will be returned. Other errors are also
// possible (e.g., if the font naming table is invalid).
func GetFamily(font *sfnt.Font) (string, error) {
	return GetProperty(font, sfnt.NameIDFamily)
}

// Returns the subfamily name of the given font. If the information
// is missing, [ErrNotFound] will be returned. Other errors are also
// possible (e.g., if the font naming table is invalid).
//
// In most cases, the subfamily value will be one of:
//  - Regular, Italic, Bold, Bold Italic
func GetSubfamily(font *sfnt.Font) (string, error) {
	return GetProperty(font, sfnt.NameIDSubfamily)
}

// Returns the name of the given font. If the information is missing,
// [ErrNotFound] will be returned. Other errors are also possible (e.g.,
// if the font naming table is invalid).
func GetName(font *sfnt.Font) (string, error) {
	return GetProperty(font, sfnt.NameIDFull)
}

// Returns the identifier of the given font. If the information is missing,
// [ErrNotFound] will be returned. Other errors are also possible (e.g.,
// if the font naming table is invalid).
func GetIdentifier(font *sfnt.Font) (string, error) {
	return GetProperty(font, sfnt.NameIDUniqueIdentifier)
}

// Returns the runes in the given text that can't be represented by the
// font. If runes are repeated in the input text, the returned slice may
// contain them multiple times too.
//
// If you load fonts dynamically, it is good practice to use this function
// to make sure that the fonts include all the glyphs that you require.
func GetMissingRunes(font *sfnt.Font, text string) ([]rune, error) {
	buffer := getSfntBuffer()
	defer releaseSfntBuffer(buffer)

	missing := make([]rune, 0)
	for _, codePoint := range text {
		index, err := font.GlyphIndex(buffer, codePoint)
		if err != nil { return missing, err }
		if index == 0 { missing = append(missing, codePoint) }
	}
	return missing, nil
}
