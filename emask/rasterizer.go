package emask

import "image"

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"


// Rasterizer is an interface for vector graphics rasterization to an
// alpha mask, allowing anyone to target it when writing a custom text
// renderer.
//
// This is interface is offered as an open alternative to Golang's
// concrete x/image/vector.Rasterizer type (as used by opentype).
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
	// glyph bounds or advances (see esizer.Sizer for that).
	Rasterize(sfnt.Segments, fixed.Point26_6) (*image.Alpha, error)

	// The cache signature returns an uint64 that can be used with glyph
	// caches in order to tell rasterizers apart. When using multiple
	// mask rasterizers within a single cache, you should make sure their
	// cache signatures are different if that's required. As a practical
	// standard, implementers of mask rasterizers are encouraged to leave
	// at least the 8 highest bits to be configurable by users through
	// the UserCfgCacheSignature interface.
	CacheSignature() uint64

	// Sets the function to be called when the Rasterizer configuration
	// or cache signature changes. This is a reserved function that only
	// a Renderer should call internally in order to connect its cache
	// handler to the rasterizer changes.
	SetOnChangeFunc(func(Rasterizer))

	// If anyone needs the following methods, let me know and we
	// can consider them...
	//NotifyFontChange(*sfnt.Font)
	//NotifySizeChange(fixed.Int26_6)
}

// See emask.Rasterizer's CacheSignature() documentation.
type UserCfgCacheSignature interface {
	// Sets the highest byte of 'signature' to the given value, like:
	//   signature = (signature & 0x00FFFFFFFFFFFFFF) | (uint64(value) << 56)
	SetHighByte(value uint8)
}

// Maybe I could export this, but it doesn't feel that relevant.
type vectorTracer interface {
	// Move to the given coordinate.
	MoveTo(fixed.Point26_6)

	// Create a segment to the given coordinate.
	LineTo(fixed.Point26_6)

	// Conic Bézier curve (also called quadratic). The first parameter
	// is the control coordinate, and the second one the final target.
	QuadTo(fixed.Point26_6, fixed.Point26_6)

	// Cubic Bézier curve. The first two parameters are the control
	// coordinates, and the third one is the final target.
	CubeTo(fixed.Point26_6, fixed.Point26_6, fixed.Point26_6)
}


// A low level method to rasterize glyph masks.
//
// Returned masks have their Rect coordinates adjusted so the mask is
// drawn at dot origin (0, 0) + the given fractional position. To draw
// it at a specific dot with a matching fractional position, translate
// the mask by (dot.X.Floor(), dot.Y.Floor()). If you don't want to
// adjust the fractional pixel position, you can call Rasterize with
// the zero-value fixed.Point26_6{}.
//
// The given drawing coordinate can be the current drawing dot, but as
// indicated above, only its fractional part will be considered.
//
// The image returned will be nil if the segments are empty or do
// not include any active lines or curves (e.g: space glyphs).
func Rasterize(outline sfnt.Segments, rasterizer Rasterizer, dot fixed.Point26_6) (*image.Alpha, error) {
	// return nil if the outline don't include lines or curves
	somethingToDraw := false
	for _, segment := range outline {
		if segment.Op != sfnt.SegmentOpMoveTo {
			somethingToDraw = true
			break
		}
	}
	if !somethingToDraw { return nil, nil }

	// obtain the fractional part of the coordinate
	// (always positive, between 0 and 0:63 [0.984375])
	fract := fixed.Point26_6 {
		X: dot.X & 0x0000003F,
		Y: dot.Y & 0x0000003F,
	}

	// rasterize the glyph outline
	return rasterizer.Rasterize(outline, fract)
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
		case sfnt.SegmentOpMoveTo: tracer.MoveTo(segment.Args[0])
		case sfnt.SegmentOpLineTo: tracer.LineTo(segment.Args[0])
		case sfnt.SegmentOpQuadTo: tracer.QuadTo(segment.Args[0], segment.Args[1])
		case sfnt.SegmentOpCubeTo: tracer.CubeTo(segment.Args[0], segment.Args[1], segment.Args[2])
		default:
			panic("unexpected segment.Op case")
		}
	}
}
