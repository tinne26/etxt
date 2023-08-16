package etxt

// Aligns tell a [Renderer] how to interpret the coordinates
// that [Renderer.Draw]() and other methods receive.
//
// More concretely: given some text, we have a text box or bounding
// rectangle that contains it. The text align specifies which part of
// that bounding box has to be aligned to the given coordinates.
// 
// For example: drawing "POOP" at (0, 0) with a centered align
// means that the center of the text box will be aligned to the
// (0, 0) coordinate. We should see the bottom half of "OP" on
// the top-left corner of our screen.
//
// See [Renderer.SetAlign]() for more details.
type Align uint8

// Returns the vertical component of the align. If the
// align is valid, the result can only be one of the
// following: [Top], [Midline], [VertCenter], [Baseline],
// [LastMidline], [LastBaseline], [Bottom].
func (self Align) Vert() Align { return alignVertBits & self }

// Returns the horizontal component of the align. If the
// align is valid, the result can only be one of the
// following: [Left], [HorzCenter], [Right].
func (self Align) Horz() Align { return alignHorzBits & self }

// Returns the result of overriding the current align with
// the non-empty components of the given align. If both
// components are defined for the given align, the result
// will be the given align itself. If only one component
// is defined, only that component will be overwritten.
// If the given align is completely empty, the value of
// the current align will be returned unmodified.
func (self Align) Adjusted(align Align) Align {
	horz := align.Horz()
	vert := align.Vert()
	if horz != 0 {
		if vert != 0 { return align }
		return horz | self.Vert()
	} else if vert != 0 {
		return self.Horz() | vert
	} else {
		return self
	}
}

// Returns a textual description of the align. For example:
//   (Top | Right).String() == "(Top | Right)"
//   (Right | Top).String() == "(Top | Right)"
//   Center.String() == "(VertCenter | HorzCenter)"
//   (Baseline | Left).String() == "(Baseline | Left)"
//   HorzCenter.String() == "(HorzCenter)"
//   Bottom.String() == "(Bottom)"
func (self Align) String() string {
	if self == 0 { return "(ZeroAlign)" }
	if self.Vert() == 0 { return "(" + self.horzString() + ")" }
	if self.Horz() == 0 { return "(" + self.vertString() + ")" }
	return "(" + self.vertString() + " | " + self.horzString() + ")"
}

func (self Align) vertString() string {
	switch self.Vert() {
	case Top: return "Top"
	case Midline: return "Midline"
	case VertCenter: return "VertCenter"
	case Baseline: return "Baseline"
	case Bottom: return "Bottom"
	case LastMidline:  return "LastMidline"
	case LastBaseline:  return "LastBaseline"
	default: return "VertUnknown"
	}
}

func (self Align) horzString() string {
	switch self.Horz() {
	case Left: return "Left"
	case HorzCenter: return "HorzCenter"
	case Right: return "Right"
	default: return "HorzUnknown"
	}
}

// Aligns have a vertical and a horizontal component. To set
// both components at once, you may use a bitwise OR, as in
// [Renderer.SetAlign](etxt.Left | etxt.Bottom). To retrieve or
// compare the individual components, avoid bitwise operations
// and use [Align.Vert]() and [Align.Horz]() instead.
const (
	// Horizontal aligns
	Left       Align = 0b0010_0000
	HorzCenter Align = 0b0100_0000
	Right      Align = 0b1000_0000

	// Vertical aligns
	Top          Align = 0b0000_0001 // top of font's ascent
	Midline      Align = 0b0000_0010 // top of xheight (rarely used)
	VertCenter   Align = 0b0000_1001 // middle of line height
	Baseline     Align = 0b0000_0100 // aligned to baseline
	Bottom       Align = 0b0000_1000 // bottom of font's descent
	LastMidline  Align = 0b0000_1010 // last Midline (if multiple lines) (rarely used)
	LastBaseline Align = 0b0000_1100 // last Baseline (if multiple lines) (rarely used)

	// TODO: consider CapLine vertical align? I know I'd use 
	//       it more than midline. It could be Top | Midline.
	//       But yeah, probably better extend va to 5 bits.
	//       Regarding LastCapLine... maybe I should leave only
	//       LastBaseline? It's the only really practical one.
	//       And it's not tragic because this can be implemented
	//       easily on the user side too.

	// Full aligns
	Center Align = HorzCenter | VertCenter
	
	alignVertBits Align = 0b0000_1111 // bit mask
	alignHorzBits Align = 0b1111_0000 // bit mask
)
// Internal note: the fact that the combinations of Bottom | Top,
// Baseline | Bottom and Midline | Bottom are valid is intentional
// and I intend to preserve it like that, despite the fact that it
// is ugly that Top | Baseline and Top | Midline don't result in
// Baseline and Midline respectively too. Perfect combinations are
// not possible, so this is almost only a cute detail.

// The renderer's [Align] defines how [Renderer.Draw]() and other operations
// interpret the coordinates passed to them. For example:
//  - If the align is set to (etxt.[Top] | etxt.[Left]), coordinates will 
//    be interpreted as the top-left corner of the box that the text needs
//    to occupy.
//  - If the align is set to (etxt.[Center]), coordinates will be
//    interpreted as the center of the box that the text needs to occupy.
//
// See [this image] for a visual explanation instead.
//
// Notice that aligns have a horizontal and a vertical component, so
// you can use [Renderer.SetAlign](etxt.[Right]) and similar to change only
// one of the components at a time. See [Align.Adjusted]() for more details.
//
// By default, the renderer's align is (etxt.[Baseline] | etxt.[Left]).
//
// [this image]: https://github.com/tinne26/etxt/blob/main/docs/img/gtxt_aligns.png
func (self *Renderer) SetAlign(align Align) {
	self.state.align = self.state.align.Adjusted(align)
}

// Returns the current align. See [Renderer.SetAlign]() documentation
// for more details on renderer aligns.
func (self *Renderer) GetAlign() Align {
	return self.state.align
}
