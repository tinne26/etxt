package emask

import "image"
import "image/color"

import "golang.org/x/image/font/sfnt"
import "golang.org/x/image/math/fixed"

// TODO: add some ArcTo method to draw quarter circles based on
//       cubic bézier curves? so we can (from 0, 0) ArcTo(0, 10, 10, 10)
//       instead of CubeTo(0, 5, 5, 10, 10, 10)

// A helper type to assist the creation of shapes that can later be
// converted to [sfnt.Segments] and rasterized with the [Rasterize]() method.
// Notice that this is actually unrelated to fonts, but once you have some
// rasterizers it's nice to have a way to play with them manually. Notice
// also that since [Rasterize]() is a CPU process, working with big shapes
// (based on their bounding rectangle) can be quite expensive.
//
// Despite what the names of the methods might lead you to believe,
// shapes are not created by "drawing lines", but rather by defining
// a set of boundaries that enclose an area. If you get unexpected
// results using shapes, come back to think about this.
//
// Shapes by themselves do not care about the direction you use to define
// the segments (clockwise/counter-clockwise), but rasterizers that use
// the segments most often do. For example, if you define two squares one
// inside the other, both in the same order (e.g: top-left to top-right,
// top-right to bottom right...) the rasterized result will be a single
// square. If you define them following opposite directions, instead,
// the result will be the difference between the two squares.
//
// [sfnt.Segments]: https://pkg.go.dev/golang.org/x/image/font/sfnt#Segments
type Shape struct {
	segments []sfnt.Segment
	invertY bool // but rasterizers already invert coords, so this is negated
}

// Creates a new Shape object. The commandsCount is used to
// indicate the initial capacity of its internal segments buffer.
func NewShape(commandsCount int) Shape {
	return Shape {
		segments: make([]sfnt.Segment, 0, commandsCount),
		invertY: false,
	}
}

// Returns whether [Shape.InvertY] is active or inactive.
func (self *Shape) HasInvertY() bool { return self.invertY }

// Let's say you want to draw a triangle pointing up, similar to an
// "A". By default, you would move to (0, 0) and then draw lines to
// (k, 2*k), (2*k, 0) and back to (0, 0).
//
// If you set InvertY to true, the previous shape will draw a triangle
// pointing down instead, similar to a "V". This is a convenient flag
// that makes it easier to work on different contexts (e.g., font glyphs
// are defined with the ascenders going into the negative y plane).
//
// InvertY can also be used creatively or to switch between clockwise and
// counter-clockwise directions when drawing symmetrical shapes that have
// their center at (0, 0).
func (self *Shape) InvertY(active bool) { self.invertY = active }

// Gets the shape information as [sfnt.Segments]. The underlying data
// is referenced both by the Shape and the sfnt.Segments, so be
// careful what you do with it.
//
// [sfnt.Segments]: https://pkg.go.dev/golang.org/x/image/font/sfnt#Segments
func (self *Shape) Segments() sfnt.Segments {
	return sfnt.Segments(self.segments)
}

// Moves the current position to (x, y).
// See [golang.org/x/image/vector.Rasterizer] operations and
// [golang.org/x/image/font/sfnt.Segment].
func (self *Shape) MoveTo(x, y int) {
	self.MoveToFract(fixed.Int26_6(x << 6), fixed.Int26_6(y << 6))
}

// Like [Shape.MoveTo], but with fixed.Int26_6 coordinates.
func (self *Shape) MoveToFract(x, y fixed.Int26_6) {
	if !self.invertY { y = -y }
	self.segments = append(self.segments,
		sfnt.Segment {
			Op: sfnt.SegmentOpMoveTo,
			Args: [3]fixed.Point26_6 {
				fixed.Point26_6{x, y},
				fixed.Point26_6{},
				fixed.Point26_6{},
			},
		})
}

// Creates a straight boundary from the current position to (x, y).
// See [golang.org/x/image/vector.Rasterizer] operations and
// [golang.org/x/image/font/sfnt.Segment].
func (self *Shape) LineTo(x, y int) {
	self.LineToFract(fixed.Int26_6(x << 6), fixed.Int26_6(y << 6))
}

// Like [Shape.LineTo], but with fixed.Int26_6 coordinates.
func (self *Shape) LineToFract(x, y fixed.Int26_6) {
	if !self.invertY { y = -y }
	self.segments = append(self.segments,
		sfnt.Segment {
			Op: sfnt.SegmentOpLineTo,
			Args: [3]fixed.Point26_6 {
				fixed.Point26_6{x, y},
				fixed.Point26_6{},
				fixed.Point26_6{},
			},
		})
}

// Creates a quadratic Bézier curve (also known as a conic Bézier curve) to
// (x, y) with (ctrlX, ctrlY) as the control point.
// See [golang.org/x/image/vector.Rasterizer] operations and
// [golang.org/x/image/font/sfnt.Segment].
func (self *Shape) QuadTo(ctrlX, ctrlY, x, y int) {
	self.QuadToFract(
		fixed.Int26_6(ctrlX << 6), fixed.Int26_6(ctrlY << 6),
		fixed.Int26_6(x     << 6), fixed.Int26_6(y     << 6))
}

// Like [Shape.QuadTo], but with fixed.Int26_6 coordinates.
func (self *Shape) QuadToFract(ctrlX, ctrlY, x, y fixed.Int26_6) {
	if !self.invertY { ctrlY, y = -ctrlY, -y }
	self.segments = append(self.segments,
		sfnt.Segment {
			Op: sfnt.SegmentOpQuadTo,
			Args: [3]fixed.Point26_6 {
				fixed.Point26_6{ctrlX, ctrlY},
				fixed.Point26_6{    x,     y},
				fixed.Point26_6{},
			},
		})
}

// Creates a cubic Bézier curve to (x, y) with (cx1, cy1) and (cx2, cy2)
// as the control points.
// See [golang.org/x/image/vector.Rasterizer] operations and
// [golang.org/x/image/font/sfnt.Segment].
func (self *Shape) CubeTo(cx1, cy1, cx2, cy2, x, y int) {
	self.CubeToFract(
		fixed.Int26_6(cx1 << 6), fixed.Int26_6(cy1 << 6),
		fixed.Int26_6(cx2 << 6), fixed.Int26_6(cy2 << 6),
		fixed.Int26_6(x   << 6), fixed.Int26_6(y   << 6))
}

// Like [Shape.CubeTo], but with fixed.Int26_6 coordinates.
func (self *Shape) CubeToFract(cx1, cy1, cx2, cy2, x, y fixed.Int26_6) {
	if !self.invertY { cy1, cy2, y = -cy1, -cy2, -y }
	self.segments = append(self.segments,
		sfnt.Segment {
			Op: sfnt.SegmentOpCubeTo,
			Args: [3]fixed.Point26_6 {
				fixed.Point26_6{cx1, cy1},
				fixed.Point26_6{cx2, cy2},
				fixed.Point26_6{  x,   y},
			},
		})
}

// Resets the shape segments. Be careful to not be holding the segments
// from [Shape.Segments]() when calling this (they may be overriden soon).
func (self *Shape) Reset() { self.segments = self.segments[0 : 0] }

// A helper method to rasterize the current shape with the default
// rasterizer. You could then export the result to a png file, e.g.:
//   file, _ := os.Create("my_ugly_shape.png")
//   _ = png.Encode(file, shape.Paint(color.White, color.Black))
//   // ...maybe even checking errors and closing the file ;)
func (self *Shape) Paint(drawColor, backColor color.Color) *image.RGBA {
	segments := self.Segments()
	if len(segments) == 0 { return nil }
	mask, err := Rasterize(segments, &DefaultRasterizer{}, fixed.P(0, 0))
	if err != nil { panic(err) } // default rasterizer doesn't return errors
	rgba := image.NewRGBA(mask.Rect)

	r, g, b, a := drawColor.RGBA()
	nrgba := color.NRGBA64 { R: uint16(r), G: uint16(g), B: uint16(b), A: 0 }
	for y := mask.Rect.Min.Y; y < mask.Rect.Max.Y; y++ {
		for x := mask.Rect.Min.X; x < mask.Rect.Max.X; x++ {
			nrgba.A = uint16((a*uint32(mask.AlphaAt(x, y).A))/255)
			rgba.Set(x, y, mixColors(nrgba, backColor))
		}
	}
	return rgba
}

// Helper method for Shape.Paint. The same as mixOverFunc
// (defined at etxt/ebiten_no.go) on the generic version of etxt.
func mixColors(draw color.Color, back color.Color) color.Color {
	dr, dg, db, da := draw.RGBA()
	if da == 0xFFFF { return draw }
	if da == 0      { return back }
	br, bg, bb, ba := back.RGBA()
	if ba == 0      { return draw }
	return color.RGBA64 {
		R: uint16N((dr*0xFFFF + br*(0xFFFF - da))/0xFFFF),
		G: uint16N((dg*0xFFFF + bg*(0xFFFF - da))/0xFFFF),
		B: uint16N((db*0xFFFF + bb*(0xFFFF - da))/0xFFFF),
		A: uint16N((da*0xFFFF + ba*(0xFFFF - da))/0xFFFF),
	}
}

// clamping from uint32 to uint16 values
func uint16N(value uint32) uint16 {
	if value > 65535 { return 65535 }
	return uint16(value)
}
