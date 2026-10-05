package etxt

import (
	"strings"
	"unicode"

	"golang.org/x/image/font/sfnt"
)

// [Gateway] to [RendererScript] functionality.
//
// [Gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.10#Renderer
func (self *Renderer) Script() *RendererScript {
	return (*RendererScript)(self)
}

// This type exists only for documentation and structuring purposes, acting as
// a [gateway] to assign fonts to scripts identified by [unicode] tables
// included in [unicode.Scripts], like [unicode.Latin], [unicode.Cyrillic],
// [unicode.Han], etc.
//
// In general, this type is used through method chaining:
//
//	renderer.Script().SetFont(unicode.Han, notoFont)
//
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt@v0.0.10#Renderer
type RendererScript Renderer

// ---- wrapper methods ----

// SetFont assigns a font to the given script, replacing any font previously
// assigned to it. Use nil to remove the assignment.
//
// Whenever script-specific fonts are configured, the renderer performs text
// itemization before measuring and drawing. If a section of the text is found
// to use a specific script for which a custom font is configured, that font
// will be used; otherwise, the primary font set through [Renderer.SetFont]()
// will prevail.
//
// Scripts must be one of the [unicode] tables included in [unicode.Scripts],
// like [unicode.Latin], [unicode.Cyrillic], [unicode.Han], etc.
//
// Multiple scripts can be associated to the same font.
func (self *RendererScript) SetFont(script *unicode.RangeTable, font *sfnt.Font) {
	(*Renderer)(self).scriptSetFont(script, font)
}

// GetFont returns the font assigned to the given script, or nil if the
// script has no font assigned.
func (self *RendererScript) GetFont(script *unicode.RangeTable) *sfnt.Font {
	return (*Renderer)(self).scriptGetFont(script)
}

// Clear removes all script-font associations.
func (self *RendererScript) Clear() {
	(*Renderer)(self).scriptClear()
}

// Each invokes fn for each script-font association configured in the renderer.
// fn can stop iteration early by returning false. Iteration is read-only.
//
// The script name is the key used in [unicode.Scripts].
func (self *RendererScript) Each(fn func(name string, script *unicode.RangeTable, font *sfnt.Font) bool) {
	(*Renderer)(self).scriptEach(fn)
}

// ---- underlying implementations ----

// scriptFont stores a script-font association
type scriptFont struct {
	name   string
	script *unicode.RangeTable
	font   *sfnt.Font
}

func (self *Renderer) scriptSetFont(script *unicode.RangeTable, font *sfnt.Font) {
	if script == nil {
		panic("nil script")
	}

	// search within existing script-font entries for fast remove/replace
	if i, found := self.scriptTableIndex(script); found {
		if self.state.scriptFonts[i].font == font {
			return // redundant replace, skip
		}
		scriptFonts := self.scriptFontsForWrite()
		if font == nil { // remove case
			copy(scriptFonts[i:], scriptFonts[i+1:])
			scriptFonts[len(scriptFonts)-1] = scriptFont{}
			self.state.scriptFonts = scriptFonts[:len(scriptFonts)-1]
		} else { // replace case
			scriptFonts[i].font = font
		}
		return
	}

	// insertion case: find script in unicode.Scripts
	name, found := scriptName(script)
	if !found {
		panic("script not found in unicode.Scripts")
	}
	if font == nil {
		return // nothing to remove, but validating first makes invalid scripts always panic
	}
	i, _ := binarySearchFunc(self.state.scriptFonts, name, func(scriptFont *scriptFont, name string) int {
		return strings.Compare(scriptFont.name, name)
	})
	scriptFonts := append(self.scriptFontsForWrite(), scriptFont{})
	copy(scriptFonts[i+1:], scriptFonts[i:])
	scriptFonts[i] = scriptFont{name, script, font}
	self.state.scriptFonts = scriptFonts
	self.state.scriptFontsBuffer = scriptFonts // append may have moved it to a larger array
}

func (self *Renderer) scriptGetFont(script *unicode.RangeTable) *sfnt.Font {
	if i, found := self.scriptTableIndex(script); found {
		return self.state.scriptFonts[i].font
	}
	return nil
}

func (self *Renderer) scriptClear() {
	scriptFonts := self.scriptFontsForWrite()
	clearScriptFonts(scriptFonts)
	self.state.scriptFonts = scriptFonts[:0]
}

func (self *Renderer) scriptEach(fn func(string, *unicode.RangeTable, *sfnt.Font) bool) {
	for i := range self.state.scriptFonts {
		scriptFont := &self.state.scriptFonts[i]
		if !fn(scriptFont.name, scriptFont.script, scriptFont.font) {
			return
		}
	}
}

// scriptFontsForWrite returns the active state's script fonts, ready to
// be modified. Right after a store they are shared with the stored state,
// so the first write copies them into the active state's own buffer.
func (self *Renderer) scriptFontsForWrite() []scriptFont {
	if !self.state.areScriptFontsWritable() {
		self.state.scriptFontsBuffer = append(self.state.scriptFontsBuffer[:0], self.state.scriptFonts...)
		self.state.scriptFonts = self.state.scriptFontsBuffer
	}
	return self.state.scriptFonts
}

// clearScriptFonts zeroes the given assignments. Buffers kept for reuse
// must not hold on to fonts that are no longer assigned.
func clearScriptFonts(scriptFonts []scriptFont) {
	for i := range scriptFonts {
		scriptFonts[i] = scriptFont{}
	}
}

// scriptTableIndex searches the given script linearly through already defined
// entries. used for fast script replaces or removals.
func (self *Renderer) scriptTableIndex(script *unicode.RangeTable) (int, bool) {
	for i := range self.state.scriptFonts {
		if self.state.scriptFonts[i].script == script {
			return i, true
		}
	}
	return 0, false
}

// scriptName returns the key under which the given table appears in
// [unicode.Scripts]. this is a slow search meant to be used  only during
// insertion of new scripts.
func scriptName(script *unicode.RangeTable) (string, bool) {
	for name, table := range unicode.Scripts {
		if table == script {
			return name, true
		}
	}
	return "", false
}
