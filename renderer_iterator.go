package etxt

import "unicode/utf8"

// Definitions of private types used to iterate strings and glyphs
// on Traverse* operations. Sometimes we iterate lines in reverse,
// so there's a bit of trickiness here and there.

// A string iterator that can be used to go through lines in regular
// order or in reverse, which is needed for some combinations of text
// direction and horizontal alignments during rendering.
type strIterator struct {
	str string
	index int
	regression int // -1 if not reverse
}

// Will return -1 if no rune is left
func (self *strIterator) Next() rune {
	if self.regression == -1 {
		codePoint, size := utf8.DecodeRuneInString(self.str[self.index:])
		if size == 0 { return -1 } // reached end
		self.index += size
		return codePoint
	} else { // reverse order
		iterPoint := (self.index - self.regression)
		if iterPoint <= 0 { return -1 } // reached end
		codePoint, size := utf8.DecodeLastRuneInString(self.str[:iterPoint])
		self.regression += size
		if self.regression >= self.index {
			self.str = self.str[self.index:]
			self.LineSlide()
		}
		return codePoint
	}
}

// used when working in reverse mode
func (self *strIterator) LineSlide() {
	self.regression = 0
	for index, codePoint := range self.str {
		if codePoint == '\n' {
			if index == 0 {
				self.index = 1 // force line break inclusion
			} else {
				self.index = index
			}
			return
		}
	}

	// reached end
	self.index = len(self.str)
}

func (self *strIterator) UntilNextLineBreak() string {
	if self.regression == -1 {
		start := self.index
		if start >= len(self.str) { return "" }
		for index, codePoint := range self.str[start:] {
			if codePoint == '\n' { return self.str[start : (start + index)] }
		}
		return self.str[start:]
	} else { // reverse order
		iterPoint := (self.index - self.regression)
		if iterPoint <= 0 { return "" } // reached end
		start := iterPoint
		curr  := start
		for curr >= 1 {
			codePoint, size := utf8.DecodeLastRuneInString(self.str[:curr])
			if codePoint == '\n' { return self.str[curr:start] }
			curr -= size
		}
		return self.str[:start]
	}
}

func newStrIterator(text string, reverse bool) strIterator {
	iter := strIterator { str: text, index: 0, regression: -1 }
	if reverse { iter.LineSlide() }
	return iter
}


type glyphsIterator struct {
	glyphs []GlyphIndex
	index int // -N if reverse
}

// The bool indicates if we already reached the end. The returned
// GlyphIndex must be ignored if bool == true
func (self *glyphsIterator) Next() (GlyphIndex, bool) {
	index  := self.index
	glyphs := self.glyphs
	if index >= 0 {
		if index >= len(glyphs) { return 0, true }
		glyphIndex := glyphs[index]
		self.index = index + 1
		return glyphIndex, false
	} else { // self.index < 0 (reverse mode)
		glyphIndex := glyphs[-index - 1]

		// update index
		if index == -1 {
			self.index = len(glyphs)
		} else {
			self.index = index + 1
		}

		return glyphIndex, false
	}
}

func newGlyphsIterator(glyphs []GlyphIndex, reverse bool) glyphsIterator {
	if !reverse { return glyphsIterator{ glyphs, 0 } }
	return glyphsIterator{ glyphs, -len(glyphs) }
}
