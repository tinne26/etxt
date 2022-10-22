package etxt

import "io/fs"
import "embed"
import "errors"
import "strings"
import "path/filepath"

// A collection of fonts accessible by name.
//
// The goal of a FontLibrary is to make it easy to load fonts in bulk
// and keep them all in a single place.
//
// FontLibrary doesn't know about system fonts, but there are other
// packages out there that can find those for you, if you are interested.
type FontLibrary struct {
	fonts map[string]*Font
}

// Creates a new, empty font library.
func NewFontLibrary() *FontLibrary {
	return &FontLibrary {
		fonts: make(map[string]*Font),
	}
}

// Returns the current number of fonts in the library.
func (self *FontLibrary) Size() int { return len(self.fonts) }

// Returns the list of fonts currently loaded in the FontLibrary as a string.
// The result includes the font name and the amount of glyphs for each font.
// Mostly useful for debugging and discovering font names and families.
// func (self *FontLibrary) StringView() string {
// 	var strBuilder strings.Builder
// 	firstFont := true
// 	for name, font := range self.fonts {
// 		if firstFont { firstFont = false } else { strBuilder.WriteRune('\n') }
// 		strBuilder.WriteString("* " + name + " (" + strconv.Itoa(font.NumGlyphs()) + " glyphs)")
// 	}
// 	return strBuilder.String()
// }

// Finds out whether a font with the given name exists in the library.
func (self *FontLibrary) HasFont(name string) bool {
	_, found := self.fonts[name]
	return found
}

// Returns the font with the given name, or nil if not found.
//
// If you don't know what are the names of your fonts, there are a few
// ways to figure it out:
//  - Load the fonts into the font library and print their names with
//    [FontLibrary.EachFont].
//  - Use the [FontName]() function directly on a [*Font] object.
//  - Open a font with the OS's default font viewer; the name is usually
//    on the title and/or first line of text.
func (self *FontLibrary) GetFont(name string) *Font {
	font, found := self.fonts[name]
	if found { return font }
	return nil
}

// Returns false if the font can't be removed due to not being found.
//
// This function is rarely necessary unless your program also has some
// mechanism to keep adding more and more fonts without restrictions.
//
// The given font name must match the name returned by the original font
// parsing function. Font names can also be recovered through
// [FontLibrary.EachFont].
func (self *FontLibrary) RemoveFont(name string) bool {
	_, found := self.fonts[name]
	if !found { return false }
	delete(self.fonts, name)
	return true
}

// Returns the name of the added font and any possible error.
// If error == nil, the font name will be non-empty.
//
// If a font with the same name has already been loaded,
// [ErrAlreadyLoaded] will be returned.
func (self *FontLibrary) ParseFontFrom(path string) (string, error) {
	font, name, err := ParseFontFrom(path)
	if err != nil { return name, err }
	return name, self.addNewFont(font, name)
}

// Similar to [FontLibrary.ParseFontFrom], but taking the font bytes
// directly. The font bytes may be gzipped. The bytes must not be
// modified while the font is in use.
func (self *FontLibrary) ParseFontBytes(fontBytes []byte) (string, error) {
	font, name, err := ParseFontBytes(fontBytes)
	if err != nil { return name, err }
	return name, self.addNewFont(font, name)
}

var ErrAlreadyLoaded = errors.New("font already loaded")
func (self *FontLibrary) addNewFont(font *Font, name string) error {
	if self.HasFont(name) { return ErrAlreadyLoaded }
	self.fonts[name] = font
	return nil
}

// Calls the given function for each font in the library, passing their
// names and content as arguments.
//
// If the given function returns a non-nil error, EachFont will immediately
// stop and return that error. Otherwise, EachFont will always return nil.
//
// Example code to print the names of all the fonts in the library:
//  fontLib.EachFont(func(name string, _ *etxt.Font) error {
//      fmt.Println(name)
//      return nil
//  })
func (self *FontLibrary) EachFont(fontFunc func(string, *Font) error) error {
	for name, font := range self.fonts {
		err := fontFunc(name, font)
		if err != nil { return err }
	}
	return nil
}

// Walks the given directory non-recursively and adds all the .ttf and .otf
// fonts in it. Returns the number of fonts added, the number of fonts skipped
// (a font with the same name already exists in the FontLibrary) and any error
// that might happen during the process.
func (self *FontLibrary) ParseDirFonts(dirName string) (int, int, error) {
	absDirPath, err := filepath.Abs(dirName)
	if err != nil { return 0, 0, err }

	loaded, skipped := 0, 0
	err = filepath.WalkDir(absDirPath,
		func(path string, info fs.DirEntry, err error) error {
			if err != nil { return err }
			if info.IsDir() {
				if path == absDirPath { return nil }
				return fs.SkipDir
			}

			valid, _ := acceptFontPath(path)
			if !valid { return nil }
			_, err = self.ParseFontFrom(path)
			if err == ErrAlreadyLoaded {
				skipped += 1
				return nil
			}
			if err == nil { loaded += 1 }
			return err
		})
	return loaded, skipped, err
}

// Same as [FontLibrary.ParseDirFonts] but for embedded filesystems.
func (self *FontLibrary) ParseEmbedDirFonts(dirName string, embedFileSys embed.FS) (int, int, error) {
	entries, err := embedFileSys.ReadDir(dirName)
	if err != nil { return 0, 0, err }

	if dirName == "." {
		dirName = ""
	} else if !strings.HasSuffix(dirName, "/") {
		dirName += "/"
	}

	loaded, skipped := 0, 0
	for _, entry := range entries {
		if entry.IsDir() { continue }
		path := dirName + entry.Name()
		valid, _ := acceptFontPath(path)
		if !valid { continue }
		_, err = self.ParseEmbedFontFrom(path, embedFileSys)
		if err == ErrAlreadyLoaded {
			skipped += 1
			continue
		}
		if err != nil { return loaded, skipped, err }
		loaded += 1
	}
	return loaded, skipped, nil
}

// Same as [FontLibrary.ParseFontFrom] but for embedded filesystems.
func (self *FontLibrary) ParseEmbedFontFrom(path string, embedFileSys embed.FS) (string, error) {
	font, name, err := ParseEmbedFontFrom(path, embedFileSys)
	if err != nil { return name, err }
	return name, self.addNewFont(font, name)
}
