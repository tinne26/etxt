package font

import "io/fs"
import "errors"
import "path/filepath"
import "golang.org/x/image/font/sfnt"

// A collection of fonts accessible by name.
//
// The goal of a library is to make it easy to parse fonts in bulk
// and keep them all in a single place.
//
// A library doesn't know about system fonts, but there are other
// packages out there that help you with that if you need it.
type Library struct {
	fonts map[string]*sfnt.Font
}

// Creates a new, empty font [Library].
func NewLibrary() *Library {
	return &Library {
		fonts: make(map[string]*sfnt.Font),
	}
}

// Returns the current number of fonts in the library.
func (self *Library) Size() int { return len(self.fonts) }

// Returns the list of fonts available in the Library as a string.
// The result includes the font name and the amount of glyphs for each font.
// Mostly useful for debugging and discovering font names and families.
// func (self *Library) StringView() string {
// 	var strBuilder strings.Builder
// 	firstFont := true
// 	for name, font := range self.fonts {
// 		if firstFont { firstFont = false } else { strBuilder.WriteRune('\n') }
// 		strBuilder.WriteString("* " + name + " (" + strconv.Itoa(font.NumGlyphs()) + " glyphs)")
// 	}
// 	return strBuilder.String()
// }

// Finds out whether a font with the given name exists in the library.
func (self *Library) HasFont(name string) bool {
	_, found := self.fonts[name]
	return found
}

// Returns the font with the given name, or nil if not found.
//
// If you don't know what are the names of your fonts, there are a few
// ways to figure it out:
//  - Add the fonts into the font library and print their names with
//    [Library.EachFont]().
//  - Use the [GetName]() function directly on a font.
//  - Open a font with the OS's default font viewer; the name is usually
//    on the title and/or first line of text.
func (self *Library) GetFont(name string) *sfnt.Font {
	font, found := self.fonts[name]
	if found { return font }
	return nil
}

// Adds the given font into the library and returns its name and any
// possible error. If the given font is nil, the method will panic. If
// another font with the same name was already present in the library,
// [ErrAlreadyPresent] will be returned.
//
// This method is rarely necessary unless the font parsing is done
// by an external package. In general, using the built-in parsing
// functions (e.g. [Library.ParseFromBytes]()) would be preferable.
func (self *Library) AddFont(font *sfnt.Font) (string, error) {
	name, err := GetName(font)
	if err != nil { return "", err }
	return name, self.addNewFont(font, name)
}

// Returns false if the font can't be removed due to not being found.
//
// This function is rarely necessary unless your program exposes font
// management directly to the user.
//
// The given font name must match the name returned by the original font
// parsing function. Font names can also be recovered through
// [Library.EachFont]().
func (self *Library) RemoveFont(name string) bool {
	_, found := self.fonts[name]
	if !found { return false }
	delete(self.fonts, name)
	return true
}

// Returns the name of the added font and any possible error.
// If error == nil, the font name will be non-empty.
//
// If a font with the same name has already been parsed or added,
// [ErrAlreadyPresent] will be returned.
func (self *Library) ParseFromPath(path string) (string, error) {
	font, name, err := ParseFromPath(path)
	if err != nil { return name, err }
	return name, self.addNewFont(font, name)
}

// The equivalent of [Library.ParseFromPath]() for raw font bytes.
// The bytes must not be modified while the font is in use. When in
// doubt, pass a copy (e.g. ParseFromBytes(append([]byte(nil), data))).
func (self *Library) ParseFromBytes(fontBytes []byte) (string, error) {
	font, name, err := ParseFromBytes(fontBytes)
	if err != nil { return name, err }
	return name, self.addNewFont(font, name)
}

// An error that can be returned by [Library.AddFont](), [Library.ParseFromPath]()
// and [Library.ParseFromBytes]() when a font is not added due to its name already
// being present in the [Library].
var ErrAlreadyPresent = errors.New("font already present in the library")

func (self *Library) addNewFont(font *sfnt.Font, name string) error {
	if self.HasFont(name) { return ErrAlreadyPresent }
	self.fonts[name] = font
	return nil
}

// Special error that can be used with [Library.EachFont]() to
// break early. When used, the function will return early but still
// return a nil error.
var ErrBreakEach = errors.New("EachFont() early break")

// Calls the given function for each font in the library, passing their
// names and content as arguments, in pseudo-random order.
//
// If the given function returns a non-nil error, the method will immediately
// stop and return that error, with the only exception of [ErrBreakEach].
// Otherwise, [Library.EachFont]() will always return nil.
//
// Example code to print the names of all the fonts in the library:
//   library.EachFont(func(name string, _ *etxt.Font) error {
//       fmt.Println(name)
//       return nil
//   })
func (self *Library) EachFont(fontFunc func(string, *sfnt.Font) error) error {
	for name, font := range self.fonts {
		err := fontFunc(name, font)
		if err != nil {
			if err == ErrBreakEach { return nil }
			return err
		}
	}
	return nil
}

// Walks the given directory non-recursively and adds all the .ttf and .otf
// fonts in it. Returns the number of fonts added, the number of fonts skipped
// (when a font with the same name already exists in the Library) and any error
// that might happen during the process.
func (self *Library) ParseAllFromPath(dirName string) (added, skipped int, err error) {
	absDirPath, err := filepath.Abs(dirName)
	if err != nil { return 0, 0, err }

	err = filepath.WalkDir(absDirPath,
		func(path string, info fs.DirEntry, err error) error {
			if err != nil { return err }
			if info.IsDir() {
				if path == absDirPath { return nil }
				return fs.SkipDir
			}

			valid := hasValidFontExtension(path)
			if !valid { return nil }
			_, err = self.ParseFromPath(path)
			if err == ErrAlreadyPresent {
				skipped += 1
				return nil
			}
			if err == nil { added += 1 }
			return err
		})
	return added, skipped, err
}

// The equivalent of [Library.ParseFromPath]() for filesystems.
// This is mainly provided to support [embed.FS] and embedded fonts.
func (self *Library) ParseFromFS(filesys fs.FS, path string) (string, error) {
	font, name, err := ParseFromFS(filesys, path)
	if err != nil { return name, err }
	return name, self.addNewFont(font, name)
}

// The equivalent of [Library.ParseAllFromPath]() for filesystems.
// This is mainly provided to support [embed.FS] and embedded fonts.
func (self *Library) ParseAllFromFS(filesys fs.FS, dirName string) (added, skipped int, err error) {
	entries, err := fs.ReadDir(filesys, dirName)
	if err != nil { return 0, 0, err }

	if dirName == "." {
		dirName = ""
	} else if len(dirName) == 0 || dirName[len(dirName) - 1] != '/' {
		dirName += "/"
	}

	for _, entry := range entries {
		if entry.IsDir() { continue }
		valid := hasValidFontExtension(entry.Name())
		if !valid { continue }
		path := dirName + entry.Name()
		_, err = self.ParseFromFS(filesys, path)
		if err == ErrAlreadyPresent {
			skipped += 1
			continue
		}
		if err != nil { return added, skipped, err }
		added += 1
	}
	return added, skipped, nil
}
