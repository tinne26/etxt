package etxt

import "github.com/tinne26/etxt/fract"

// Related to [Twine.PushEffectWithSpacing](). When logical sizes are used, the 
// values will be considered to use a base size of 16 pixels. So, if the renderer's
// font scaled size is 16, the logical values will match the scaled values. If the
// renderer's font scaled size is 32, the logical values will be multiplied by 2
// to obtain the scaled values.
// 
// Regarding line wrapping and paddings, the following rules are applied
// (actually, twine draw with wrap is unimplemented, but...):
//  - PrePad + LineBreakPad will be preferently applied.
//  - If there's not enough space, the effect will be moved directly
//    to the next line, using LineStartPad + PostPad.
//  - If there's not enough space, LineStartPad + LineBreakPad will
//    be used, even if this ends up exceeding the maximum line wrap
//    width. But in this case, no more content will be added after
//    the line break pad, even if something could fit afterwards.
//  - MinWidth will always be respected even if ends up leading to
//    a LineStartPad + LineBreakPad situation that still overflows
//    the maximum line wrap width.
type TwineEffectSpacing struct {
	PrePad       fract.Unit
	PostPad      fract.Unit
	MinWidth     fract.Unit // can't be disconnected from PrePad or LineStartPad
	LineStartPad fract.Unit // should be <= PrePad
	LineBreakPad fract.Unit // should be <= PostPad
	ArePadsLogical    bool // if true, all units are considered as if they were on size 16
	IsMinWidthLogical bool // if true, all units are considered as if they were on size 16
}

func (self *TwineEffectSpacing) parseFromData(data []byte) int {
	// read control byte first
	if len(data) == 0 { panic("invalid twine effect spacing encoding: missing control byte") }
	control := data[0]
	self.ArePadsLogical    = ((control & 0b1000_0000) != 0)
	self.IsMinWidthLogical = ((control & 0b0100_0000) != 0)

	parts := (control & 0b0011_1111)
	if parts > 5 { panic("invalid control byte") }
	numDataBytes := int(parts*3 + 1)
	if len(data) < numDataBytes { panic("invalid twine effect spacing encoding (missing parts)") }
	switch parts {
	case 1:
		self.MinWidth = fractFromBytes(data[1], data[2], data[3])
		self.PrePad, self.PostPad = 0, 0
		self.LineStartPad, self.LineBreakPad = 0, 0
	case 2:
		self.MinWidth = 0
		self.PrePad, self.PostPad = fractFromBytes(data[1], data[2], data[3]), fractFromBytes(data[4], data[5], data[6])
		self.LineStartPad, self.LineBreakPad = 0, 0
	case 3:
		self.MinWidth = fractFromBytes(data[1], data[2], data[3])
		self.PrePad, self.PostPad = fractFromBytes(data[4], data[5], data[6]), fractFromBytes(data[7], data[8], data[9])
		self.LineStartPad, self.LineBreakPad = 0, 0
	case 4:
		self.MinWidth = 0
		self.PrePad, self.PostPad = fractFromBytes(data[1], data[2], data[3]), fractFromBytes(data[4], data[5], data[6])
		self.LineStartPad, self.LineBreakPad = fractFromBytes(data[7], data[8], data[9]), fractFromBytes(data[10], data[11], data[12])
	case 5:
		self.MinWidth = fractFromBytes(data[1], data[2], data[3])
		self.PrePad, self.PostPad = fractFromBytes(data[4], data[5], data[6]), fractFromBytes(data[7], data[8], data[9])
		self.LineStartPad, self.LineBreakPad = fractFromBytes(data[10], data[11], data[12]), fractFromBytes(data[13], data[14], data[15])
	default:
		panic("broken code")
	}
	return numDataBytes
}

func (self *TwineEffectSpacing) appendData(buffer []byte) []byte {
	var control byte
	if self.ArePadsLogical    { control |= 0b1000_0000 }
	if self.IsMinWidthLogical { control |= 0b0100_0000 }
	hasMinWidth := (self.MinWidth != 0)
	hasPads := (self.PrePad != 0 || self.PostPad != 0)
	hasLinePads := (self.LineStartPad != 0 || self.LineBreakPad != 0)

	if hasLinePads {
		pre1, pre2, pre3 := fractToBytes(self.PrePad)
		pst1, pst2, pst3 := fractToBytes(self.PostPad)
		ls1, ls2, ls3 := fractToBytes(self.LineStartPad)
		lb1, lb2, lb3 := fractToBytes(self.LineBreakPad)
		if hasMinWidth {
			mw1, mw2, mw3 := fractToBytes(self.MinWidth)
			return append(buffer, []byte{control | 5, mw1, mw2, mw3, pre1, pre2, pre3, pst1, pst2, pst3, ls1, ls2, ls3, lb1, lb2, lb3}...)
		} else {
			return append(buffer, []byte{control | 4, pre1, pre2, pre3, pst1, pst2, pst3, ls1, ls2, ls3, lb1, lb2, lb3}...)
		}
	} else if hasPads {
		pre1, pre2, pre3 := fractToBytes(self.PrePad)
		pst1, pst2, pst3 := fractToBytes(self.PostPad)
		if hasMinWidth {
			mw1, mw2, mw3 := fractToBytes(self.MinWidth)
			return append(buffer, []byte{control | 3, mw1, mw2, mw3, pre1, pre2, pre3, pst1, pst2, pst3}...)
		} else {
			return append(buffer, []byte{control | 2, pre1, pre2, pre3, pst1, pst2, pst3}...)
		}
	} else if hasMinWidth {
		mw1, mw2, mw3 := fractToBytes(self.MinWidth)
		return append(buffer, []byte{control | 1, mw1, mw2, mw3}...)
	} else {
		panic("attempted to append empty TwineEffectSpacing")
	}
}

func (self *TwineEffectSpacing) getPrePad(scaledSize fract.Unit) fract.Unit {
	if !self.ArePadsLogical { return self.PrePad }
	return self.PrePad.Rescale(16 << 6, scaledSize)
}

func (self *TwineEffectSpacing) getPostPad(scaledSize fract.Unit) fract.Unit {
	if !self.ArePadsLogical { return self.PostPad }
	return self.PostPad.Rescale(16 << 6, scaledSize)
}

func (self *TwineEffectSpacing) getLineStartPad(scaledSize fract.Unit) fract.Unit {
	if !self.ArePadsLogical { return self.LineStartPad }
	return self.LineStartPad.Rescale(16 << 6, scaledSize)
}

func (self *TwineEffectSpacing) getLineBreakPad(scaledSize fract.Unit) fract.Unit {
	if !self.ArePadsLogical { return self.LineBreakPad }
	return self.LineBreakPad.Rescale(16 << 6, scaledSize)
}

func (self *TwineEffectSpacing) getMinWidth(scaledSize fract.Unit) fract.Unit {
	if !self.IsMinWidthLogical { return self.MinWidth }
	return self.MinWidth.Rescale(16 << 6, scaledSize)
}
