package etxt

import "github.com/tinne26/etxt/fract"

func (self *Renderer) DrawWithWrap(target TargetImage, text string, x, y, widthLimit int) {
	self.fractDrawWithWrap(target, text, fract.FromInt(x), fract.FromInt(y), widthLimit)
}

func (self *Renderer) fractDrawWithWrap(target TargetImage, text string, x, y fract.Unit, widthLimit int) {
	panic("unimplemented") // TODO
}
