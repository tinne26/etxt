package emask

// A common buffer implementation shared by edge_marker and outliner.
type buffer struct {
	Width  int // canvas width, in pixels
	Height int // canvas height, in pixels
	Values []float64
	// ^ Negative values are used for counter-clockwise segments,
	//   positive values are used for clockwise segments.
}

// Sets a new Width and Height and resizes the underlying buffer
// if necessary. The buffer contents are always cleared too.
func (self *buffer) Resize(width, height int) {
	if width <= 0 || height <= 0 { panic("width or height <= 0") }
	self.Width  = width
	self.Height = height
	totalLen := width*height
	if len(self.Values) == totalLen {
		// nothing
	} else if len(self.Values) > totalLen {
		self.Values = self.Values[0 : totalLen]
	} else { // len(self.Values) < totalLen
		if cap(self.Values) >= totalLen {
			self.Values = self.Values[0 : totalLen]
		} else {
			self.Values = make([]float64, totalLen)
			return // stop before ClearBuffer()
		}
	}

	self.Clear()
}

// Fills the internal buffer with zeros.
func (self *buffer) Clear() { fastFillFloat64(self.Values, 0) }

// Performs the boundary change accumulation operation storing
// the results into the given buffer. Used with the edge marker
// rasterizer.
func (self *buffer) AccumulateUint8(buffer []uint8) {
	if len(buffer) != self.Width*self.Height {
		panic("uint8 buffer has wrong length")
	}

	index := 0
	for y := 0; y < self.Height; y++ {
		accumulator := float64(0)
		accUint8    := uint8(0)
		for x := 0; x < self.Width; x++ {
			value := self.Values[index]
			if value != 0 { // small optimization
				accumulator += value
				accUint8 = uint8(clampUnit64(abs64(accumulator))*255)
			}
			buffer[index] = accUint8
			index += 1
		}
	}
}
