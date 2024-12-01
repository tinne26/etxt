package etxt

import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/sizer"
import "github.com/tinne26/etxt/fract"

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
	fontIndex   FontIndex
	blendMode   BlendMode
}
