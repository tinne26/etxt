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
	activeFont *sfnt.Font

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
