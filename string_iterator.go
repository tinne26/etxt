package etxt

import "unicode/utf8"

// Definitions of private types used to iterate strings and glyphs
// on Traverse* operations. Sometimes we iterate lines in reverse,
// so there's a bit of trickiness here and there.

type ltrStringIterator struct {
	nextRuneStart int
	prevRuneStart int
}

func (self *ltrStringIterator) Next(text string) rune {
	if self.nextRuneStart < len(text) {
		codePoint, runeSize := utf8.DecodeRuneInString(text[self.nextRuneStart:])
		self.prevRuneStart = self.nextRuneStart
		self.nextRuneStart += runeSize
		return codePoint
	} else {
		return -1
	}
}

func (self *ltrStringIterator) PeekNext(text string) rune {
	if self.nextRuneStart < len(text) {
		codePoint, _ := utf8.DecodeRuneInString(text[self.nextRuneStart:])
		return codePoint
	} else {
		return -1
	}
}

func (self *ltrStringIterator) Unroll(codePoint rune) {
	self.nextRuneStart -= utf8.RuneLen(codePoint)
}

func (self *ltrStringIterator) StringLeft(text string) string {
	if self.nextRuneStart >= len(text) {
		return ""
	}
	return text[self.nextRuneStart:]
}

type rtlStringIterator struct {
	head, tail    int
	nextRuneEnd   int
	prevRuneStart int
}

func (self *rtlStringIterator) Init(text string) {
	self.tail = 0
	self.head = 0
	self.LineSlide(text)
}

func (self *rtlStringIterator) LineSlide(text string) {
	self.tail = self.head
	if self.head >= len(text) {
		self.nextRuneEnd = self.tail
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
		self.nextRuneEnd = self.head
	}
}

func (self *rtlStringIterator) Next(text string) rune {
	if self.nextRuneEnd > self.tail {
		codePoint, runeSize := utf8.DecodeLastRuneInString(text[:self.nextRuneEnd])
		self.nextRuneEnd -= runeSize
		self.prevRuneStart = self.nextRuneEnd
		if codePoint == '\n' || self.nextRuneEnd <= self.tail {
			self.LineSlide(text)
		}
		return codePoint
	} else {
		return -1
	}
}

func (self *rtlStringIterator) PeekNext(text string) rune {
	if self.nextRuneEnd > self.tail {
		codePoint, _ := utf8.DecodeLastRuneInString(text[:self.nextRuneEnd])
		return codePoint
	} else {
		return -1
	}
}
