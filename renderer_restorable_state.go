package etxt

import "image/color"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/mask"
import "github.com/tinne26/etxt/sizer"
import "github.com/tinne26/etxt/fract"

type restorableState struct {
	fontColor color.Color
	fontSizer sizer.Sizer
	rasterizer mask.Rasterizer
	fonts []*sfnt.Font

	textDirection Direction
	horzQuantization uint8
	vertQuantization uint8
	align Align

	scale fract.Unit
	logicalSize fract.Unit
	scaledSize fract.Unit
	fontIndex uint8
	
	blendMode BlendMode
	//_ uint8
	//_ uint8
}
