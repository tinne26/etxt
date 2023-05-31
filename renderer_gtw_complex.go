package etxt

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

// This type exists only for documentation and structuring purposes,
// acting as a [gateway] to access advanced [Renderer] properties and
// operating directly with glyphs and the more flexible [Text] type.
//
// These types and features are mainly relevant when working with
// rich text, [complex scripts] and text shaping.
//
// In general, this type is used through method chaining:
//   renderer.Complex().Draw(canvas, text, x, y)
//
// [complex scripts]: https://github.com/tinne26/etxt/blob/main/docs/shaping.md
// [gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
type RendererComplex Renderer

// [Gateway] to [RendererComplex] functionality.
//
// [Gateway]: https://pkg.go.dev/github.com/tinne26/etxt#Renderer
func (self *Renderer) Complex() *RendererComplex {
	return (*RendererComplex)(self)
}

// Sets the text direction to be used on subsequent operations.
//
// Do not confuse text direction with horizontal align. Text
// direction is typically only changed for right-to-left languages
// like Arabic, Hebrew or Persian.
//
// By default, the direction is [LeftToRight].
func (self *RendererComplex) SetDirection(dir Direction) {
	(*Renderer)(self).complexSetDirection(dir)
}

// Returns the current main text direction. See [RendererComplex.SetDirection]()
// for more details.
func (self *RendererComplex) GetDirection() Direction {
	return (*Renderer)(self).complexGetDirection()
}

// Sets the renderer's fonts. Having multiple fonts is mostly relevant
// when working with rich text, for example when trying to combine regular,
// bold and italic subfamilies in the same draw call.
//
// To change the main font statically, use [RendererComplex.SetFontIndex]().
func (self *RendererComplex) SetFonts(fonts []*sfnt.Font) {
	(*Renderer)(self).complexSetFonts(fonts)
}

// Returns the renderer fonts available for use with rich text.
func (self *RendererComplex) GetFonts() []*sfnt.Font {
	return (*Renderer)(self).fonts
}

// Sets the active font index to the given value.
// Returns the newly active font.
func (self *RendererComplex) SetFontIndex(index uint8) *sfnt.Font {
	return (*Renderer)(self).complexSetFontIndex(index)
}

// Returns the index of the renderer's main font.
func (self *RendererComplex) GetFontIndex() int {
	return int((*Renderer)(self).fontIndex)
}

// Same as [RendererFract.Draw](), but expecting a [Text] instead of a string.
func (self *RendererComplex) Draw(screen TargetImage, text Text, x, y fract.Unit) {
	panic("unimplemented / TODO")
	// TODO: consider Drawi or Drawf() for the fractional versions?
}

// Same as [RendererFract.DrawWithWrap](), but expecting a [Text] instead of a string.
func (self *RendererComplex) DrawWithWrap(screen TargetImage, text Text, x, y fract.Unit, widthLimit int) {
	panic("unimplemented / TODO")
}

// ---- implementations ----

func (self *Renderer) complexSetDirection(dir Direction) {
	// basically, this can change the text iteration order,
	// from first \n to next, to next \n to first.
	switch dir {
	case LeftToRight:
		self.internalFlags &= ^internalFlagDirIsRTL
	case RightToLeft:
		self.internalFlags |= internalFlagDirIsRTL
	default:
		panic("invalid direction")
	}
}

func (self *Renderer) complexGetDirection() Direction {
	if self.internalFlags & internalFlagDirIsRTL != 0 {
		return RightToLeft
	} else {
		return LeftToRight
	}
}

func (self *Renderer) complexSetFonts(fonts []*sfnt.Font) {
	oldFont := self.GetFont()
	self.fonts = fonts
	newFont := self.GetFont()
	if oldFont != newFont {
		self.notifyFontChange(newFont)
	}
}

func (self *Renderer) complexSetFontIndex(index uint8) *sfnt.Font {
	// change active font index and notify change if relevant
	oldFont := self.GetFont()
	self.fontIndex = uint8(index)
	newFont := self.GetFont()
	if oldFont != newFont {
		self.notifyFontChange(newFont)
	}

	// return the new selected font
	return newFont
}

// func (self *Renderer) complexSetFont(font *sfnt.Font, key uint8) {
// 	ikey := int(key)
// 	if len(self.fonts) <= ikey {
// 		if cap(self.fonts) <= ikey {
// 			newFonts := make([]*sfnt.Font, ikey + 1)
// 			copy(newFonts, self.fonts)
// 			self.fonts = newFonts
// 		} else {
// 			self.fonts = self.fonts[ : ikey + 1]
// 		}
// 	}

// 	self.fonts[ikey] = font
// }
