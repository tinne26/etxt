package mask

import "image"

import "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/fract"


// Rasterizer is an interface for 2D vector graphics rasterization to an
// alpha mask. This interface is offered as an open alternative to the
// concrete [golang.org/x/image/vector.Rasterizer] type (as used by
// [golang.org/x/image/font/opentype]), allowing anyone to target it and
// use its own rasterizer for text rendering.
//
// Mask rasterizers can't be used concurrently and must tolerate
// coordinates out of bounds.
type Rasterizer interface {
	// Rasterizes the given outline to an alpha mask. The outline must be
	// drawn at the given fractional position (always positive coords between
	// 0 and 0:63 (= 0.984375)).
	//
	// Notice that rasterizers might create masks bigger than Segments.Bounds()
	// to account for their own special effects, but they still can't affect
	// glyph bounds or advances (see sizer.Sizer for that).
	Rasterize(sfnt.Segments, fract.Point) (*image.Alpha, error)

	// The signature returns a uint64 that can be used with glyph caches
	// in order to tell rasterizers apart. When using multiple mask
	// rasterizers with a single cache, you normally want to make sure
	// that their signatures are different.
	Signature() uint64

	// Sets the function to be called when the Rasterizer configuration
	// or the signature change. This is a reserved function that only a
	// Renderer should call internally in order to connect its cache
	// handler to the rasterizer changes.
	SetOnChangeFunc(func(Rasterizer))

	// If anyone needs the following methods, let me know and we
	// can consider them...
	//NotifyFontChange(*sfnt.Font)
	//NotifySizeChange(fract.Unit)
}

// Maybe I could export this, but it doesn't feel that relevant.
type vectorTracer interface {
	// Move to the given coordinate.
	MoveTo(fract.Point)

	// Create a segment to the given coordinate.
	LineTo(fract.Point)

	// Conic Bézier curve (also called quadratic). The first parameter
	// is the control coordinate, and the second one the final target.
	QuadTo(fract.Point, fract.Point)

	// Cubic Bézier curve. The first two parameters are the control
	// coordinates, and the third one is the final target.
	CubeTo(fract.Point, fract.Point, fract.Point)
}

// A low level method to rasterize glyph masks.
//
// Returned masks have their coordinates adjusted so the mask is drawn at
// dot origin (0, 0) + the given fractional position by default. To draw it at
// a specific dot with a matching fractional position, translate the mask by
// dot.X.Floor() and dot.Y.Floor(). If you don't want to adjust the fractional
// pixel position, you can call Rasterize with a zero-value fract.Point{}.
//
// The given drawing coordinate can be your current drawing dot, but as
// indicated above, only its fractional part will be considered.
//
// The image returned will be nil if the segments are empty or do
// not include any active lines or curves (e.g.: space glyphs).
func Rasterize(outline sfnt.Segments, rasterizer Rasterizer, dot fract.Point) (*image.Alpha, error) {
	// return nil if the outline don't include lines or curves
	for _, segment := range outline {
		if segment.Op == sfnt.SegmentOpMoveTo { continue }
		return rasterizer.Rasterize(outline, dot)
	}
	return nil, nil // nothing to draw
}

// Calls MoveTo(), LineTo(), QuadTo() and CubeTo() methods on the
// tracer, as corresponding, for each segment in the glyph outline.
// This could also be placed on emask.Rasterize, but I decided
// to keep it separate and pass the outline directly to the rasterizer
// instead, as some advanced use-cases might benefit from having the
// segments. Feel free to copy-paste if writing your own glyph mask
// rasterizer.
func processOutline(tracer vectorTracer, outline sfnt.Segments) {
	for _, segment := range outline {
		switch segment.Op {
		case sfnt.SegmentOpMoveTo:
			tracer.MoveTo(
				fract.Point{X: fract.Unit(segment.Args[0].X), Y: fract.Unit(segment.Args[0].Y)},
			)
		case sfnt.SegmentOpLineTo:
			tracer.LineTo(
				fract.Point{X: fract.Unit(segment.Args[0].X), Y: fract.Unit(segment.Args[0].Y)},
			)
		case sfnt.SegmentOpQuadTo:
			tracer.QuadTo(
				fract.Point{X: fract.Unit(segment.Args[0].X), Y: fract.Unit(segment.Args[0].Y)},
				fract.Point{X: fract.Unit(segment.Args[1].X), Y: fract.Unit(segment.Args[1].Y)},
			)
		case sfnt.SegmentOpCubeTo:
			tracer.CubeTo(
				fract.Point{X: fract.Unit(segment.Args[0].X), Y: fract.Unit(segment.Args[0].Y)},
				fract.Point{X: fract.Unit(segment.Args[1].X), Y: fract.Unit(segment.Args[1].Y)},
				fract.Point{X: fract.Unit(segment.Args[2].X), Y: fract.Unit(segment.Args[2].Y)},
			)
		default:
			panic("unexpected segment.Op case")
		}
	}
}
