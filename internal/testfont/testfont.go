// Package testfont lets tests request the fonts they need by glyph coverage,
// out of the fonts placed in a test directory.
//
// It doesn't depend on etxt/font, so the tests of that package can use it too.
package testfont

import (
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"golang.org/x/image/font/sfnt"
)

// Font is a test font and the file it was parsed from.
type Font struct {
	Font *sfnt.Font
	Name string
	File string // file name within the directory
}

// Dir is a directory of test fonts, parsed the first time they are needed.
type Dir struct {
	fsys fs.FS
	path string

	once  sync.Once
	fonts []Font
	err   error

	mutex   sync.Mutex
	skipped []string // tests skipped for missing fonts
}

// NewDir returns the test fonts at the given path of the filesystem.
func NewDir(fsys fs.FS, path string) *Dir {
	return &Dir{fsys: fsys, path: path}
}

// Filter is used for [Dir.Matching].
type Filter struct {
	Has   string
	Lacks string
}

// String returns the filter's runes, for skip messages.
func (self Filter) String() string {
	if self.Lacks == "" {
		return fmt.Sprintf("%q", self.Has)
	}
	return fmt.Sprintf("%q without %q", self.Has, self.Lacks)
}

func (self Filter) matches(font *sfnt.Font) bool {
	var buffer sfnt.Buffer
	for _, codePoint := range self.Has {
		if index, err := font.GlyphIndex(&buffer, codePoint); err != nil || index == 0 {
			return false
		}
	}
	for _, codePoint := range self.Lacks {
		if index, err := font.GlyphIndex(&buffer, codePoint); err == nil && index != 0 {
			return false
		}
	}
	return true
}

// WithRunes maps to [Dir.Matching] with only Has filters.
func (self *Dir) WithRunes(tb testing.TB, samples ...string) []Font {
	tb.Helper()
	filters := make([]Filter, len(samples))
	for i := range samples {
		filters[i].Has = samples[i]
	}
	return self.Matching(tb, filters...)
}

// Matching returns a different test font for each of the given filters.
// If fonts can't be loaded, the test fails. If suitable fonts can't be found,
// the test is skipped and recorded for ReportSkips. On success, the returned
// fonts are also logged.
func (self *Dir) Matching(tb testing.TB, filters ...Filter) []Font {
	tb.Helper()
	picked, found := pick(self.All(tb), filters)
	if !found {
		self.mutex.Lock()
		self.skipped = append(self.skipped, tb.Name())
		self.mutex.Unlock()
		tb.Skipf("not enough test fonts for %v (see test/README.md)", filters)
	}

	names := make([]string, len(picked))
	for i := range picked {
		names[i] = picked[i].Name
	}
	tb.Logf("test fonts: %s", strings.Join(names, ", "))
	return picked
}

// ReportSkips is used by TestMain to make test runs fail if any test was
// skipped due to missing fonts. This is useful to prevent all tests being
// silently skipped when using typical test commands.
func (self *Dir) ReportSkips(exitCode int) int {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if exitCode != 0 || len(self.skipped) == 0 {
		return exitCode
	}
	fmt.Printf("missing test fonts, skipped: %s (see test/README.md)\n", strings.Join(self.skipped, ", "))
	return 1
}

// All returns all the test fonts, sorted by name, without repeated names.
// If the ETXT_TEST_SEED environment variable is set, they are shuffled with
// that seed instead, so tests pick other fonts in a reproducible way. It
// fails the test if the fonts can't be loaded.
func (self *Dir) All(tb testing.TB) []Font {
	tb.Helper()
	self.once.Do(func() { self.fonts, self.err = load(self.fsys, self.path) })
	if self.err != nil {
		tb.Fatalf("failed to load test fonts: %s", self.err)
	}
	return self.fonts
}

func load(fsys fs.FS, path string) ([]Font, error) {
	entries, err := fs.ReadDir(fsys, path)
	if err != nil {
		return nil, err
	}

	var fonts []Font
	var buffer sfnt.Buffer
	names := make(map[string]bool)
	for _, entry := range entries {
		file := entry.Name()
		if !strings.HasSuffix(file, ".ttf") && !strings.HasSuffix(file, ".otf") {
			continue
		}
		data, err := fs.ReadFile(fsys, path+"/"+file)
		if err != nil {
			return nil, err
		}
		font, err := sfnt.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		name, err := font.Name(&buffer, sfnt.NameIDFull)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if names[name] {
			continue
		}
		names[name] = true
		fonts = append(fonts, Font{font, name, file})
	}
	sort.Slice(fonts, func(i, j int) bool {
		return fonts[i].Name < fonts[j].Name
	})

	if seed, found := os.LookupEnv("ETXT_TEST_SEED"); found {
		value, err := strconv.ParseInt(seed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("ETXT_TEST_SEED must be an integer, got %q", seed)
		}
		rng := rand.New(rand.NewSource(value))
		rng.Shuffle(len(fonts), func(i, j int) {
			fonts[i], fonts[j] = fonts[j], fonts[i]
		})
	}
	return fonts, nil
}

// pick returns a different font for each filter, trying the fonts in order.
// When the remaining filters can't be satisfied, it tries the next font for
// the first one.
func pick(fonts []Font, filters []Filter) ([]Font, bool) {
	if len(filters) == 0 {
		return nil, true
	}
	for i, font := range fonts {
		if !filters[0].matches(font.Font) {
			continue
		}
		others := append(append([]Font{}, fonts[:i]...), fonts[i+1:]...)
		if rest, found := pick(others, filters[1:]); found {
			return append([]Font{font}, rest...), true
		}
	}
	return nil, false
}
