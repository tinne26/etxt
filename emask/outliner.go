package emask

import "math"

type outliner struct {
	x float64
	y float64
	thickness float64 // can't be modified throughout an outline
	marginFactor uint8
	Buffer buffer
	CurveSegmenter curveSegmenter

	segments [5]outlineSegment // the 0 and 1 are kept for closing
	openSegmentCount int
}

// Sets the thickness of the outliner. Thickness can only be
// modified while not drawing. This means it can only be changed
// after MoveTo, ClosePath, CutPath or after initialization but
// before any LineTo, QuadTo or CubeTo commands are issued.
//
// This method will panic if any of the previous conditions are
// violated or if the passed thickness is zero, negative or bigger
// than 1024.
//
// The thickness will be quantized to a multiple of 1/1024 and
// the quantized value will be returned.
func (self *outliner) SetThickness(thickness float64) float64 {
	if self.openSegmentCount > 0 {
		panic("can't change thickness while drawing")
	}
	if thickness <= 0 { panic("thickness <= 0 not allowed") }
	if thickness > 1024 { panic("thickness > 1024 not allowed") }
	self.thickness = math.Round(thickness*1024)/1024
	if self.thickness == 0 { self.thickness = 1/1024 }
	return self.thickness
}

// TODO: probably needs to be quantized too, but I don't think everything
//       will fit in the uint64 anyway. say 10 bits thickness, 10 bits
//       margin factor, 8 bits signature, then curve segmenter needs
//       40 bits... so, 68 bits already... and thickness needs actually
//       more like 20 bits. say 20 thick, 20 curve, 8 sig, 10 margin,
//       8 curve splits... that's still 66 bits. margin in 8 bits would
//       be impossible I think. 18 for curve and 18 for thickness may be
//       possible, but really tricky. Well, I can do it in 0.5 parts, up to
//       128. or 0.1 parts up to 25.6. that's not insane. to be seen if
//       curve quantization in 20 bits is enough...
func (self *outliner) SetMarginFactor(factor float64) {
	if factor < 1.0  { panic("outliner margin factor must be >= 1.0" ) }
	if factor > 16.0 { panic("outliner margin factor must be <= 16.0") }
	self.marginFactor = uint8(math.Round((factor - 1.0)*16))
}

func (self *outliner) MaxMargin() float64 {
	return self.thickness*(float64(self.marginFactor) + 1.0)/16
}

// Moves the current position to the given coordinates.
func (self *outliner) MoveTo(x, y float64) {
	if self.openSegmentCount > 0 {
		self.CutPath() // cut previous path if not closed yet
	}
	self.x = x
	self.y = y
}

// Creates a straight line from the current position to the given
// target with the current thickness and moves the current position
// to the new one.
func (self *outliner) LineTo(x, y float64) {
	if self.x == x && self.y == y { return }
	defer func(){ self.x, self.y = x, y }()

	// compute new line ax + by + c = 0 coefficients
	dx := x - self.x
	dy := y - self.y
	c  := dx*self.y - dy*self.x
	a, b, c := toLinearFormABC(self.x, self.y, x, y)

	// if the new line goes in the same direction as the
	// previous one, do not add it as a new line
	if self.openSegmentCount > 0 {
		prevSegment := &self.segments[self.openSegmentCount - 1]
		xdiv := prevSegment.a*b - a*prevSegment.b
		if xdiv <= 0.00001 && xdiv >= -0.00001 {
			prevSegment.fx = x
			prevSegment.fy = y

			start := self.segments[0] // check if closing outline
			if start.ox == x && start.oy == y { self.ClosePath() }
			return
		}
	}

	// find parallels at the given distance that will delimit the new segment
	c1, c2 := parallelsAtDist(a, b, c, self.thickness/2)

	// create the segment
	self.segments[self.openSegmentCount] = outlineSegment{
		ox: self.x, oy: self.y, fx: x, fy: y,
		a: a, b: b, c1: c1, c2: c2,
	}
	self.openSegmentCount += 1
	switch self.openSegmentCount {
	case 3: // fill segment 1
		self.segments[1].Fill(&self.Buffer, &self.segments[0], &self.segments[2])
	case 4: // fill segment 2
		self.segments[2].Fill(&self.Buffer, &self.segments[1], &self.segments[3])
	case 5: // fill one segment and remove another old one
		self.segments[3].Fill(&self.Buffer, &self.segments[2], &self.segments[4])
		self.segments[2] = self.segments[3]
		self.segments[3] = self.segments[4]
		self.openSegmentCount = 4
	}

	// see if we are closing the outline
	if self.openSegmentCount > 1 {
		start := self.segments[0]
		if start.ox == x && start.oy == y { self.ClosePath() }
	}
}

// Creates a boundary from the current position to the given target
// as a quadratic Bézier curve through the given control point and
// moves the current position to the new one.
func (self *outliner) QuadTo(ctrlX, ctrlY, fx, fy float64) {
	self.CurveSegmenter.TraceQuad(self.LineTo, self.x, self.y, ctrlX, ctrlY, fx, fy)
}

// Creates a boundary from the current position to the given target
// as a cubic Bézier curve through the given control points and
// moves the current position to the new one.
func (self *outliner) CubeTo(cx1, cy1, cx2, cy2, fx, fy float64) {
	self.CurveSegmenter.TraceCube(self.LineTo, self.x, self.y, cx1, cy1, cx2, cy2, fx, fy)
}

// Closes a path without tying back to the starting point.
func (self *outliner) CutPath() {
	switch self.openSegmentCount {
	case 0: return // superfluous call
	case 1: // cut both head and tail
		self.segments[0].Cut(&self.Buffer)
	default: // cut start tail, cut end head
		sc := self.openSegmentCount
		self.segments[0].CutTail(&self.Buffer, &self.segments[1])
		self.segments[sc - 1].CutHead(&self.Buffer, &self.segments[sc - 2])
	}
	self.openSegmentCount = 0
}

// Closes a path tying back to the starting point (if possible).
func (self *outliner) ClosePath() {
	sc := self.openSegmentCount
	if sc <= 2 {
		self.CutPath()
	} else {
		self.segments[     0].Fill(&self.Buffer, &self.segments[sc - 1], &self.segments[1])
		self.segments[sc - 1].Fill(&self.Buffer, &self.segments[sc - 2], &self.segments[0])
	}
	self.openSegmentCount = 0
}
