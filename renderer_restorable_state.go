package etxt

import (
	"image/color"

	"github.com/tinne26/etxt/fract"
	"github.com/tinne26/etxt/mask"
	"github.com/tinne26/etxt/sizer"
	"golang.org/x/image/font/sfnt"
)

type restorableState struct {
	fontColor  color.Color
	fontSizer  sizer.Sizer
	rasterizer mask.Rasterizer

	font *sfnt.Font // the primary font, see also Renderer.activeFont

	scriptFonts       []scriptFont // sorted by name. owned when its backing array is scriptFontsBuffer's
	scriptFontsBuffer []scriptFont // this depth's own memory, kept in its slot across restores

	textDirection    Direction
	horzQuantization uint8
	vertQuantization uint8
	align            Align

	scale       fract.Unit
	logicalSize fract.Unit
	scaledSize  fract.Unit
	fontIndex   fontIndex
	blendMode   BlendMode
}

func (self *restorableState) hasScriptFonts() bool {
	return len(self.scriptFonts) > 0
}

// areScriptFontsWritable reports whether scriptFonts can be modified in place.
// It's false right after a store, when the active state shares its script
// fonts with the stored one and changing them would change both.
func (self *restorableState) areScriptFontsWritable() bool {
	if cap(self.scriptFonts) == 0 || cap(self.scriptFontsBuffer) == 0 {
		return false // a slice without capacity has no memory
	}
	// slices share memory when their first elements have the same address.
	// [:1] works on empty slices too, as long as they have capacity
	return &self.scriptFonts[:1][0] == &self.scriptFontsBuffer[:1][0]
}
