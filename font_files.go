package etxt

import "os"
import "io"
import "io/fs"
import "errors"
import "strings"
import "path/filepath"
import "compress/gzip"
import "embed"

import "golang.org/x/image/font/sfnt"

// Parses a font and returns it along its name and any possible
// error. Supported formats are .ttf, .otf, .ttf.gz and .otf.gz.
//
// This is a low level function; you may prefer to use FontLibrary
// instead.
func ParseFontFrom(path string) (*Font, string, error) {
	// check font path validity
	knownFontExt, gzipped := acceptFontPath(path)
	if !knownFontExt {
		return nil, "", errors.New("invalid font path '" + path + "'")
	}

	// open font file
	file, err := os.Open(path)
	if err != nil { return nil, "", err }
	return parseFontFileAndClose(file, gzipped)
}

// Same as ParseFontFrom, but for embedded filesystems.
//
// This is a low level function; you may prefer to use FontLibrary
// instead.
func ParseEmbedFontFrom(path string, embedFileSys *embed.FS) (*Font, string, error) {
	// check font path validity
	knownFontExt, gzipped := acceptFontPath(path)
	if !knownFontExt {
		return nil, "", errors.New("invalid font path '" + path + "'")
	}

	// open font file
	file, err := embedFileSys.Open(path)
	if err != nil { return nil, "", err }
	return parseFontFileAndClose(file, gzipped)
}

func parseFontFileAndClose(file io.ReadCloser, gzipped bool) (*Font, string, error) {
	fileCloser := onceCloser{ file, false }
	defer fileCloser.Close()

	// detect gzipping
	var reader io.ReadCloser
	var readerCloser *onceCloser
	if gzipped {
		gzipReader, err := gzip.NewReader(file)
		if err != nil { return nil, "", err }
		reader = gzipReader
		readerCloser = &onceCloser{ gzipReader, false }
		defer readerCloser.Close()
	} else {
		reader = file
		readerCloser = &fileCloser
	}

	// read font bytes
	bytes, err := io.ReadAll(reader)
	if err != nil { return nil, "", err }
	err = readerCloser.Close()
	if err != nil { return nil, "", err }
	err = fileCloser.Close()
	if err != nil { return nil, "", err }

	// create font from bytes and get name
	return ParseFontBytes(bytes)
}

// Same as [sfnt.Parse], but also including the font name.
// The bytes must not be modified while the font is in use.
//
// This is a low level function; you may prefer to use FontLibrary
// instead.
//
// [sfnt.Parse]: https://pkg.go.dev/golang.org/x/image/font/sfnt#Parse.
func ParseFontBytes(bytes []byte) (*Font, string, error) {
	newFont, err := sfnt.Parse(bytes)
	if err != nil { return nil, "", err }
	fontName, err := FontName(newFont)
	return newFont, fontName, err
}

// Applies GzipFontFile to each font of the given directory.
func GzipDirFonts(fontsDir string, outputDir string) error {
	absDirPath, err := filepath.Abs(fontsDir)
	if err != nil { return err }
	absOutDir,  err := filepath.Abs(outputDir)
	if err != nil { return err }

	return filepath.WalkDir(absDirPath,
		func(path string, info fs.DirEntry, err error) error {
			if err != nil { return err }
			if info.IsDir() {
				if path == absDirPath { return nil }
				return fs.SkipDir
			}

			knownFontExt, gzipped := acceptFontPath(path)
			if knownFontExt && !gzipped {
				err := GzipFontFile(path, absOutDir)
				if err != nil { return err }
			}
			return nil
		})
}

// Compresses the given font by gzipping it and storing the result on
// outDir with the same name as the original but an extra .gz extension.
//
// The font size reduction can vary a lot depending on the font and format,
// but it's typically above 33%, with many .ttf font sizes being halved.
//
// If you are wondering why gzip is used instead of supporting .woff formats:
// gzip has stdlib support, can be applied transparently, and compression rates
// are very similar to what brotli achieves for .woff files.
//
// On linux systems, when working on games, in many cases it's easier to
// simply compress once with a `gzip -k your_font.ttf` command instead of
// using this library.
func GzipFontFile(fontPath string, outDir string) error {
	// make output dir if it doesn't exist yet
	info, err := os.Stat(outDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) { return err }
	if err != nil { // must be fs.ErrNotExist
		err = os.Mkdir(outDir, 0700)
		if err != nil { return err }
	} else if !info.IsDir() {
		return errors.New("'" + outDir + "' is not a directory")
	}

	if !strings.HasSuffix(outDir, string(os.PathSeparator)) {
		outDir += string(os.PathSeparator)
	}

	// open font file
	fontFile, err := os.Open(fontPath)
	if err != nil { return err }
	fontFileCloser := onceCloser{ fontFile, false }
	defer fontFileCloser.Close()

	// create new compressed font file
	gzipFile, err := os.Create(outDir + filepath.Base(fontPath) + ".gz")
	if err != nil { return err }
	gzipFileCloser := onceCloser{ gzipFile, false }
	defer gzipFileCloser.Close()

	// write new compressed file
	gzipWriter := gzip.NewWriter(gzipFile) // DefaultCompression is perfectly ok
	gzipWriterCloser := onceCloser{ gzipWriter, false }
	defer gzipWriterCloser.Close()
	_, err = io.Copy(gzipWriter, fontFile)
	if err != nil { return err }

	// close everything that can be closed
	err = gzipWriterCloser.Close()
	if err != nil { return err }
	err = gzipFileCloser.Close()
	if err != nil { return err }
	err = fontFileCloser.Close()
	if err != nil { return err }
	return nil
}

// --- helpers ---

// onceCloser makes it easier to both defer closes (to cover for early error
// returns) and check close errors manually when done with other operations,
// without having to suffer from "file already closed" and similar issues.
type onceCloser struct { closer io.Closer ; alreadyClosed bool }
func (self *onceCloser) Close() error {
	if self.alreadyClosed { return nil }
	self.alreadyClosed = true
	return self.closer.Close()
}

// The first bool returns whether to accept the font path or not.
// The second indicates if the font is gzipped or not.
func acceptFontPath(path string) (bool, bool) {
	gzipped := false
	if strings.HasSuffix(path, ".gz") {
		gzipped = true
		path = path[0 : len(path) - 3]
	}

	validExt := (strings.HasSuffix(path, ".ttf") || strings.HasSuffix(path, ".otf"))
	return validExt, gzipped
}
