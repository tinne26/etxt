package etxt

import "strconv"
import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/sizer"
import "github.com/tinne26/etxt/mask"

// Registers a new restore point for the renderer's state. You can jump
// back to the previously memorized point with [RendererUtils.Rewind]().
// 
// The memorized state includes the following properties:
//  - Align, color, size, scale, blend mode, main font, rasterizer,
//    sizer, quantization and text direction.
// Notably, custom rendering function, secondary fonts and cache
// handler are not memorized.
//
// For improved safety when using renderer state restore points, you
// may also want to consider [RendererUtils.AssertMaxRestorePoints]().
func (self *RendererUtils) Memorize() {
	(*Renderer)(self).utilsMemorize()
}

// Jumps back to the previous restore point created through
// [RendererUtils.Memorize]() and removes it after restoring
// the renderer's state.
//
// If the restore points stack is empty and no rewinding is
// possible, this function will return false.
func (self *RendererUtils) Rewind() bool {
	return (*Renderer)(self).utilsRewind()
}

// Panics when the size of the restore points stack exceeds
// the given value. Restore points are created through
// [RendererUtils.Memorize]().
// 
// There are two main ways to use this function:
//  - Regularly asserting that the number of restore points
//    stay below a value that you never expect to reach (e.g.,
//    64) in order to prevent memory leaks.
//  - Passing zero to the function to ensure the restore points
//    stack is completely empty in those points where you
//    expect this to be the case.
func (self *RendererUtils) AssertMaxRestorePoints(n int) {
	if n > len(self.stateRestorePoints) {
		givenMax  := strconv.Itoa(n)
		actualMax := strconv.Itoa(len(self.stateRestorePoints))
		panic("expected at most " + givenMax + " restore points, found " + actualMax)
	}
}

type stateSnapshot struct {
	fontColor color.Color
	fontSizer sizer.Sizer
	rasterizer mask.Rasterizer
	mainFont *sfnt.Font

	horzQuantization uint8
	vertQuantization uint8
	fontIndex uint8
	align Align

	scale fract.Unit
	logicalSize fract.Unit
	ltr bool
	blendMode BlendMode
}

func (self *Renderer) utilsMemorize() {
	// ensure relevant properties are initialized
	if self.missingBasicProps() { self.initBasicProps() }
	self.initSizer()
	self.initRasterizer()

	snapshot := stateSnapshot{}
	snapshot.fontColor = self.fontColor
	snapshot.fontSizer = self.fontSizer
	snapshot.rasterizer = self.rasterizer
	snapshot.mainFont = self.GetFont()
	snapshot.horzQuantization = self.horzQuantization
	snapshot.vertQuantization = self.vertQuantization
	snapshot.fontIndex = self.fontIndex
	snapshot.align = self.align
	snapshot.blendMode = self.blendMode
	snapshot.ltr = ((self.internalFlags & internalFlagDirIsRTL) == 0)
	snapshot.scale = self.scale
	snapshot.logicalSize = self.logicalSize
}

func (self *Renderer) utilsRewind() bool {
	if len(self.stateRestorePoints) == 0 { return false }
	snapshot := self.stateRestorePoints[len(self.stateRestorePoints) - 1]

	self.fontColor = snapshot.fontColor
	font := self.GetFont()
	self.fontIndex = snapshot.fontIndex
	if font != snapshot.mainFont {
		self.fonts = ensureSliceSize(self.fonts, int(snapshot.fontIndex) + 1)
		self.fonts[snapshot.fontIndex] = font
		self.notifyFontChange(snapshot.mainFont)
	}

	if self.logicalSize != snapshot.logicalSize || self.scale != snapshot.scale {
		self.logicalSize = snapshot.logicalSize
		self.scale = snapshot.scale
		self.refreshScaledSize()
	}

	if snapshot.fontSizer != self.fontSizer {
		self.fontSizer = snapshot.fontSizer
		self.fontSizer.NotifyChange(snapshot.mainFont, &self.buffer, self.scaledSize)
	}
	
	if snapshot.rasterizer != self.rasterizer {
		self.glyphSetRasterizer(snapshot.rasterizer)
	}
	
	self.horzQuantization = snapshot.horzQuantization
	self.vertQuantization = snapshot.vertQuantization
	self.align = snapshot.align
	self.blendMode = snapshot.blendMode
	if snapshot.ltr {
		self.internalFlags = self.internalFlags & ^internalFlagDirIsRTL
	} else {
		self.internalFlags = self.internalFlags | internalFlagDirIsRTL
	}

	self.stateRestorePoints = self.stateRestorePoints[0 : len(self.stateRestorePoints) - 1]
	return true
}
