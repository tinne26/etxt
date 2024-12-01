package etxt

import "unicode/utf8"

// Definitions of private types used to iterate strings and glyphs
// on Traverse* operations. Sometimes we iterate lines in reverse,
// so there's a bit of trickiness here and there.

type ltrStringIterator struct{ index int }

func (self *ltrStringIterator) Next(text string) rune {
	if self.index < len(text) {
		codePoint, runeSize := utf8.DecodeRuneInString(text[self.index:])
		self.index += runeSize
		return codePoint
	} else {
		return -1
	}
}

func (self *ltrStringIterator) PeekNext(text string) rune {
	if self.index < len(text) {
		codePoint, _ := utf8.DecodeRuneInString(text[self.index:])
		return codePoint
	} else {
		return -1
	}
}

func (self *ltrStringIterator) Unroll(codePoint rune) {
	self.index -= utf8.RuneLen(codePoint)
}

func (self *ltrStringIterator) StringLeft(text string) string {
	if self.index >= len(text) {
		return ""
	}
	return text[self.index:]
}

type rtlStringIterator struct{ head, tail, index int }

func (self *rtlStringIterator) Init(text string) {
	self.tail = 0
	self.head = 0
	self.LineSlide(text)
}

func (self *rtlStringIterator) LineSlide(text string) {
	self.tail = self.head
	if self.head >= len(text) {
		self.index = self.tail
	} else {
		if text[self.head] == '\n' {
			self.head += 1
		} else {
			for self.head < len(text) { // find next line break or end of string
				codePoint, runeSize := utf8.DecodeRuneInString(text[self.head:])
				if codePoint == '\n' {
					break
				}
				self.head += runeSize
			}
		}
		self.index = self.head
	}
}

func (self *rtlStringIterator) Next(text string) rune {
	if self.index > self.tail {
		codePoint, runeSize := utf8.DecodeLastRuneInString(text[:self.index])
		self.index -= runeSize
		if codePoint == '\n' || self.index <= self.tail {
			self.LineSlide(text)
		}
		return codePoint
	} else {
		return -1
	}
}

func (self *rtlStringIterator) PeekNext(text string) rune {
	if self.index > self.tail {
		codePoint, _ := utf8.DecodeLastRuneInString(text[:self.index])
		return codePoint
	} else {
		return -1
	}
}
