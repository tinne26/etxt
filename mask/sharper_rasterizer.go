package mask

import "image"

import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/etxt/fract"

var _ Rasterizer = (*SharperRasterizer)(nil)

// A variant of SharpRasterizer, with more complex evaluations of what
// should be or not be a solid pixel. This process is executed on the
// CPU; an ideal implementation for Ebitengine would perform a similar
// process through a shader instead. As of right now, this is offered
// mostly as a proof of concept.
type SharperRasterizer struct{ DefaultRasterizer }

func max2(a, b uint8) uint8 {
	if a >= b {
		return a
	}
	return b
}

func max3(a, b, c uint8) uint8 {
	return max2(max2(a, b), c)
}

func stemPick(center, sideA, sideB uint8) uint8 {
	if sideA == 0 {
		return max2(center, sideB)
	}
	if sideB == 0 {
		return max2(center, sideA)
	}
	return center
}

func cornerPick(a, b uint8) uint8 {
	if a == 0 && b == 0 {
		return 255
	}
	return max2(a, b)
}

// Satisfies the [Rasterizer] interface.
func (self *SharperRasterizer) Rasterize(outline sfnt.Segments, origin fract.Point) (*image.Alpha, error) {
	mask, err := self.DefaultRasterizer.Rasterize(outline, origin)
	if err != nil {
		return mask, err
	}
	self.sharpen(mask)
	return mask, nil
}

func (self *SharperRasterizer) sharpen(mask *image.Alpha) {
	// first pass, correcting corners
	bounds := mask.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var i int = 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := mask.Pix[i]
			if value == 0 || value == 255 {
				i += 1
				continue
			}

			// check neighbours
			var up, down, left, right uint8
			if y > 0 {
				up = mask.Pix[i-width]
			}
			if y < height-1 {
				down = mask.Pix[i+width]
			}
			if x > 0 {
				left = mask.Pix[i-1]
			}
			if x < width-1 {
				right = mask.Pix[i+1]
			}

			if up == 255 {
				if left == 255 {
					mask.Pix[i] = cornerPick(right, down)
				} else if right == 255 {
					mask.Pix[i] = cornerPick(left, down)
				} else {
					mask.Pix[i] = stemPick(value, left, right)
				}
			} else if left == 255 {
				if down == 255 {
					mask.Pix[i] = cornerPick(up, right)
				} else {
					mask.Pix[i] = stemPick(value, up, down)
				}
			} else if right == 255 {
				if down == 255 { // corner case
					mask.Pix[i] = cornerPick(up, left)
				} else {
					mask.Pix[i] = stemPick(value, up, down)
				}
			} else if down == 255 {
				mask.Pix[i] = stemPick(value, left, right)
			} else {
				// isolated fragment cases. I don't know if this is necessary in practice
				// ok, this should only be done if a solid interior exists
				if up >= 128 {
					if left >= 128 && mask.Pix[i-width-1] >= 128 {
						mask.Pix[i] = max3(value, up, left)
					} else if right >= 128 && mask.Pix[i-width+1] >= 128 {
						mask.Pix[i] = max3(value, up, right)
					}
				} else if left >= 128 {
					if down >= 128 && mask.Pix[i+width-1] >= 128 {
						mask.Pix[i] = max3(value, left, down)
					}
				} else if right >= 128 {
					if down >= 128 && mask.Pix[i+width+1] >= 128 {
						mask.Pix[i] = max3(value, right, down)
					}
				}
			}

			i += 1
		}
	}

	for i := 0; i < len(mask.Pix); i++ {
		if mask.Pix[i] < 128 {
			mask.Pix[i] = 0
		} else {
			mask.Pix[i] = 255
		}
	}
}

// Satisfies the [Rasterizer] interface.
func (self *SharperRasterizer) Signature() uint64 {
	return 0x009E000000000000
}
