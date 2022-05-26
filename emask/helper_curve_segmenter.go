package emask

// A small struct to handle Bézier curve segmentation into straight
// lines. It has a configurable curveThreshold and a maxCurveSplits
// limit. Used by edge_marker and outliner.
type curveSegmenter struct {
	curveThreshold float32 // threshold to decide if a segment approximates
	                       // a bézier curve well enough or we should split
	maxCurveSplits uint8   // a cutoff for curve segmentation
}

// Sets the threshold distance to use when splitting Bézier curves into
// linear segments. If a linear segment misses the curve by more than
// the threshold value, the curve will be split. Otherwise, the linear
// segment will be used to approximate it.
//
// Values very close to zero could prevent algorithms from converging
// due to floating point instability, but MaxCurveSplits will still
// prevent infinite loops.
func (self *curveSegmenter) SetThreshold(dist float32) {
	self.curveThreshold = dist
}

// Sets the maximum amount of times a curve can be recursively split
// into subsegments while trying to approximate it with TraceQuad or
// TraceCube.
//
// The maximum number of segments that will approximate a curve is
// 2^maxCurveSplits.
//
// This value is typically used as a cutoff to prevent low curve thresholds
// from making the curve splitting process too slow, but it can also be used
// creatively to get jaggy results instead of smooth curves.
//
// Values outside the [0, 255] range will be silently clamped.
func (self *curveSegmenter) SetMaxSplits(maxCurveSplits int) {
	if maxCurveSplits < 0 {
		self.maxCurveSplits = 0
	} else if maxCurveSplits > 255 {
		self.maxCurveSplits = 255
	} else {
		self.maxCurveSplits = uint8(maxCurveSplits)
	}
}

type traceFunc = func(x, y float64) // called for each segment during curve segmentation
func (self *curveSegmenter) TraceQuad(lineTo traceFunc, x, y, ctrlX, ctrlY, fx, fy float64) {
	self.recursiveTraceQuad(lineTo, x, y, ctrlX, ctrlY, fx, fy, 0)
}

func (self *curveSegmenter) recursiveTraceQuad(lineTo traceFunc, x, y, ctrlX, ctrlY, fx, fy float64, depth uint8) (float64, float64) {
	if depth >= self.maxCurveSplits || self.withinThreshold(x, y, fx, fy, ctrlX, ctrlY) {
		lineTo(fx, fy)
		return fx, fy
	}

	ocx, ocy := lerp(x, y, ctrlX, ctrlY, 0.5)   // origin to control
	cfx, cfy := lerp(ctrlX, ctrlY, fx, fy, 0.5) // control to end
	ix , iy  := lerp(ocx, ocy, cfx, cfy, 0.5)   // interpolated point
	x, y = self.recursiveTraceQuad(lineTo, x, y, ocx, ocy, ix, iy, depth + 1)
	return self.recursiveTraceQuad(lineTo, x, y, cfx, cfy, fx, fy, depth + 1)
}

func (self *curveSegmenter) TraceCube(lineTo traceFunc, x, y, cx1, cy1, cx2, cy2, fx, fy float64) {
	self.recursiveTraceCube(lineTo, x, y, cx1, cy1, cx2, cy2, fx, fy, 0)
}

func (self *curveSegmenter) recursiveTraceCube(lineTo traceFunc, x, y, cx1, cy1, cx2, cy2, fx, fy float64, depth uint8) (float64, float64) {
	if depth >= self.maxCurveSplits || (self.withinThreshold(x, y, cx2, cy2, cx1, cy1) && self.withinThreshold(cx1, cy1, fx, fy, cx2, cy2)) {
		lineTo(fx, fy)
		return fx, fy
	}

	oc1x , oc1y  := lerp(x, y, cx1, cy1, 0.5)           // origin to control 1
	c1c2x, c1c2y := lerp(cx1, cy1, cx2, cy2, 0.5)       // control 1 to control 2
	c2fx , c2fy  := lerp(cx2, cy2, fx, fy, 0.5)         // control 2 to end
	iox  , ioy   := lerp(oc1x, oc1y, c1c2x, c1c2y, 0.5) // first interpolation from origin
	ifx  , ify   := lerp(c1c2x, c1c2y, c2fx, c2fy, 0.5) // second interpolation to end
	ix   , iy    := lerp(iox, ioy, ifx, ify, 0.5)       // cubic interpolation
	x, y = self.recursiveTraceCube(lineTo, x, y, oc1x, oc1y, iox, ioy, ix, iy, depth + 1)
	return self.recursiveTraceCube(lineTo, x, y, ifx, ify, c2fx, c2fy, fx, fy, depth + 1)
}

func (self *curveSegmenter) withinThreshold(ox, oy, fx, fy, px, py float64) bool {
	// https://en.wikipedia.org/wiki/Distance_from_a_point_to_a_line#Line_defined_by_an_equation
	// dist = |a*x + b*y + c| / sqrt(a^2 + b^2)
	a, b, c := toLinearFormABC(ox, oy, fx, fy)
	n := a*px + b*py + c
	return n*n <= float64(self.curveThreshold)*float64(self.curveThreshold)*(a*a + b*b)
}
