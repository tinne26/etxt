package etxt

// Aligns are used to define how to interpret the coordinates passed
// to renderer operations. For example, if you try to draw text at
// (0, 0) with any align that's not top-left, the result is going to
// be clipped or not visible at all.
//
// See [Renderer.SetAlign]() for more details.
type Align uint8

// Returns the vertical component of the align. If the
// align is valid, the result can only be one of the
// following: [Top], [TopBaseline], [YCenter], [BottomBaseline],
// [Bottom].
func (self Align) Vert() Align { return alignVertBits & self }

// Returns the horizontal component of the align. If the
// align is valid, the result can only be one of the
// following: [Left], [XCenter], [Right].
func (self Align) Horz() Align { return alignHorzBits & self }

// Align constants for renderer operations. Vertical
// and horizontal aligns can be combined with a bitwise
// OR (|).
const (
	Left    Align = 0b0010_0000 // horizontal align
	XCenter Align = 0b0100_0000 // horizontal align
	Right   Align = 0b1000_0000 // horizontal align

	Top            Align = 0b0000_0001 // vertical align
	TopBaseline    Align = 0b0000_0010 // vertical align
	YCenter        Align = 0b0000_0100 // vertical align
	BottomBaseline Align = 0b0000_1000 // vertical align
	Bottom         Align = 0b0001_0000 // vertical align

	Center Align = XCenter | YCenter // full align
	
	alignVertBits Align = 0b0001_1111 // bit mask
	alignHorzBits Align = 0b1110_0000 // bit mask
)

// The renderer's align defines how [Renderer.Draw]() and other operations
// interpret the coordinates passed to them. For example:
//  - If the align is set to (etxt.[Top] | etxt.[Left]), coordinates will 
//    be interpreted as the top-left corner of the box that the text needs
//    to occupy.
//  - If the align is set to (etxt.[Center]), coordinates will be
//    interpreted as the center of the box that the text needs to occupy.
//
// See [this image] for a visual explanation instead.
//
// Notice that aligns have separate horizontal and vertical components, so
// you can use calls like [Renderer.SetAlign](etxt.[Right]) to change only
// one of the components (the horizontal one in this case).
//
// By default, the renderer's align is (etxt.[Baseline] | etxt.[Left]).
//
// [this image]: https://github.com/tinne26/etxt/blob/main/docs/img/gtxt_aligns.png
func (self *Renderer) SetAlign(align Align) {
	if align == 0 { panic("invalid zero align") }
	if self.missingBasicProps() { self.initBasicProps() }
	
	// configure horizontal align
	horzAlign := align.Horz()
	if horzAlign != 0 {
		switch horzAlign {
		case Left, XCenter, Right:
			self.align = horzAlign | (self.align & alignVertBits)
		default:
			panic("invalid horizontal component in align")
		}
	}

	// configure vertical align
	vertAlign := align.Vert()
	if vertAlign != 0 {
		switch vertAlign {
		case Top, TopBaseline, YCenter, BottomBaseline, Bottom:
			self.align = vertAlign | (self.align & alignHorzBits)
		default:
			panic("invalid vertical component in align")
		}
	}
}

// Returns the current align. See [Renderer.SetAlign]() documentation
// for more details on renderer aligns.
func (self *Renderer) GetAlign() Align {
	if self.missingBasicProps() { self.initBasicProps() }
	return self.align
}
