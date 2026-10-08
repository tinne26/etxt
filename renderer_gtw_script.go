package etxt

import (
	"strings"
	"unicode"
	"unicode/utf8"

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
// Whenever script-specific fonts are configured, the renderer checks text
// scripts while measuring and drawing. If a section of the text is found to
// use a specific script for which a custom font is configured, that font is
// used; otherwise, the primary font from [Renderer.SetFont]() prevails.
//
// Spaces, ASCII digits and punctuation belong to the [Common] script, which
// generally inherit the font of the preceding text.
//
// Line metrics always come from the primary font.
//
// Scripts must be one of the writing systems in [unicode.Scripts], like
// [unicode.Latin], [unicode.Cyrillic], [unicode.Han], etc.
//
// Multiple scripts can be associated to the same font.
//
// [Common]: https://www.unicode.org/reports/tr24/#Common
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
	if script == unicode.Common || script == unicode.Inherited {
		panic("Common and Inherited can't have script-specific fonts") // not writing systems
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

// scriptSwitch marks the start of a run of text that uses a different font.
type scriptSwitch struct {
	index int // byte index of the first rune of the run
	font  *sfnt.Font
}

// openBracket is an opening bracket waiting for its closing pair.
type openBracket struct {
	closing rune       // the closing bracket that pairs with it
	font    *sfnt.Font // font of the text before it, nil at the start of a line
}

// activatePrimaryFont makes the primary font active again if it was changed
// during draw or measure while using script fonts
func (self *Renderer) activatePrimaryFont() {
	if self.state.activeFont != self.state.primaryFont {
		self.state.activeFont = self.state.primaryFont
		self.notifyFontChange(self.state.primaryFont)
	}
}

// itemizeScripts finds the fonts for the given text before drawing or
// measuring it with script fonts, and stores where they change.
//
// Common and Inherited runes, like spaces, digits and punctuation, take the
// font of the text before them. At the start of a line they take the font of
// the text after them, and recognized closing brackets take the font of their
// opening bracket. Lines without runes of their own script continue with the
// font of the previous lines, or take the first font in the text if there's
// none. Only line breaks end lines here: wrapping happens later, and doesn't
// change fonts, as wrap points depend on the widths of the fonts.
func (self *Renderer) itemizeScripts(text string) {
	switches := self.scriptSwitches[:0]
	var brackets [16]openBracket // open brackets of the current line
	var depth int
	var bracketsFull bool // more brackets were nested than fit, so pairing stopped for the line
	var lineStart int
	var lineHasScript bool // whether the line has a rune of its own script yet
	for index, codePoint := range text {
		// line breaks: brackets aren't paired across lines
		if codePoint == '\n' {
			lineStart, lineHasScript, depth, bracketsFull = index+1, false, 0, false
			continue
		}

		// Common and Inherited runes keep the current font, except closing
		// brackets, which take the font of their opening bracket
		font := self.getRuneFont(codePoint)
		if font == nil {
			if closing := closingBracket(codePoint); closing != 0 {
				// opening bracket: record the font of the text before it. If it
				// doesn't fit, its closing bracket would pair with an outer one,
				// so pairing stops for the rest of the line
				if depth == len(brackets) {
					depth, bracketsFull = 0, true
				}
				if !bracketsFull {
					brackets[depth] = openBracket{closing: closing}
					if lineHasScript {
						brackets[depth].font = switches[len(switches)-1].font
					}
					depth += 1
				}
				continue // keeps the current font, like other Common runes
			}
			for i := depth - 1; i >= 0; i-- {
				if brackets[i].closing != codePoint {
					continue // not this bracket's pair, try an outer one
				}
				if brackets[i].font != nil { // nil at the start of a line, which takes the font after it
					switches = setFontFrom(switches, index, brackets[i].font)
				}
				depth = i // also drops unclosed brackets inside this pair
				break
			}
			continue // keeps the current font, unless switched above
		}

		// runes with their own script: on the first one of a line, the
		// Common runes at the start of the line take its font too, and so
		// does the start of the text when it's the first one in the text
		if !lineHasScript {
			start := lineStart
			if len(switches) == 0 {
				start = 0
			}
			switches = setFontFrom(switches, start, font)
			for i := range brackets[:depth] {
				brackets[i].font = font
			}
			lineHasScript = true
		}
		switches = setFontFrom(switches, index, font)
	}
	if len(switches) == 0 {
		switches = append(switches, scriptSwitch{0, self.state.primaryFont})
	}
	self.scriptSwitches = switches
	self.scriptCursor = 0
}

// setFontFrom makes the given font start at the given index. It doesn't add a
// switch if that font is already the one in use.
func setFontFrom(switches []scriptSwitch, index int, font *sfnt.Font) []scriptSwitch {
	last := len(switches) - 1
	if last >= 0 && switches[last].font == font {
		return switches
	}
	if last >= 0 && switches[last].index == index {
		switches[last].font = font
		return switches
	}
	return append(switches, scriptSwitch{index, font})
}

// updateScriptFont sets the font for the rune at text[index], and reports
// whether the active font changed. Traversals move through the text in order
// or line by line backwards, so the switch is found in a few steps.
func (self *Renderer) updateScriptFont(index int) bool {
	switches, i := self.scriptSwitches, self.scriptCursor
	for i+1 < len(switches) && switches[i+1].index <= index {
		i += 1
	}
	for switches[i].index > index {
		i -= 1
	}
	self.scriptCursor = i

	font := switches[i].font
	if font == self.state.activeFont {
		return false
	}
	self.state.activeFont = font
	self.notifyFontChange(font)
	return true
}

// getRuneFont returns the font assigned to the rune's script, or the primary
// font when there's none. It returns nil for runes of the Common and Inherited
// scripts, like spaces, ASCII digits, punctuation, etc., which are contextual
func (self *Renderer) getRuneFont(codePoint rune) *sfnt.Font {
	for i := range self.state.scriptFonts {
		if unicode.Is(self.state.scriptFonts[i].script, codePoint) {
			return self.state.scriptFonts[i].font
		}
	}
	if codePoint < utf8.RuneSelf { // only ASCII letters aren't Common
		if ('a' <= codePoint && codePoint <= 'z') || ('A' <= codePoint && codePoint <= 'Z') {
			return self.state.primaryFont
		}
		return nil
	}
	if unicode.Is(unicode.Common, codePoint) || unicode.Is(unicode.Inherited, codePoint) {
		return nil
	}
	return self.state.primaryFont
}

// closingBracket returns the closing bracket for the given opening bracket,
// or 0 if the rune isn't one of the common ASCII, East Asian or mathematical
// opening brackets supported by the itemizer.
func closingBracket(codePoint rune) rune {
	switch codePoint {
	// ASCII
	case '(':
		return ')'
	case '[':
		return ']'
	case '{':
		return '}'
	// full width forms
	case '（':
		return '）'
	case '［':
		return '］'
	case '｛':
		return '｝'
	case '｟':
		return '｠'
	case '｢':
		return '｣'
	// CJK
	case '「':
		return '」'
	case '『':
		return '』'
	case '《':
		return '》'
	case '〈':
		return '〉'
	case '【':
		return '】'
	case '〔':
		return '〕'
	case '〖':
		return '〗'
	case '〘':
		return '〙'
	case '〚':
		return '〛'
	// mathematical
	case '⌈':
		return '⌉'
	case '⌊':
		return '⌋'
	case '⟨':
		return '⟩'
	case '⟦':
		return '⟧'
	case '⟪':
		return '⟫'
	}
	return 0
}
