// A collection of helper functions for examining certain font or glyph
// properties, irrelevant unless you are really deep into this mess.
//
// This subpackage is *not* used in etxt itself.
package emetric

import "golang.org/x/image/math/fixed"
import "golang.org/x/image/font/sfnt"

// cboxObox computes two bounding boxes for the given segments:
//  - The [control box], equivalent to sfnt.Segments.Bounds().
//  - The "ON" contour points bounding box.
// These two can be used by CBoxBadness to determine if the CBox
// matches the real bounding box or not (though the actual bounding
// box can't be easily determined if the two are different).
//
// [control box]: https://freetype.org/freetype2/docs/glyphs/glyphs-6.html#section-2
func cboxObox(segments sfnt.Segments) (fixed.Rectangle26_6, fixed.Rectangle26_6) {
	// create boxes
	cbox := fixed.Rectangle26_6 {
		Min: fixed.Point26_6 {
			X: fixed.Int26_6(0x7FFFFFFF),
			Y: fixed.Int26_6(0x7FFFFFFF),
		},
		Max: fixed.Point26_6 {
			X: fixed.Int26_6(-0x80000000),
			Y: fixed.Int26_6(-0x80000000),
		},
	}
	obox := fixed.Rectangle26_6 {
		Min: fixed.Point26_6 { X: cbox.Min.X, Y: cbox.Min.Y },
		Max: fixed.Point26_6 { X: cbox.Max.X, Y: cbox.Max.Y },
	}

	// iterate segments
	for _, segment := range segments {
		switch segment.Op {
		case sfnt.SegmentOpMoveTo, sfnt.SegmentOpLineTo:
			adjustBoxLimits(&cbox, segment.Args[0 : 1])
			adjustBoxLimits(&obox, segment.Args[0 : 1])
		case sfnt.SegmentOpQuadTo:
			adjustBoxLimits(&cbox, segment.Args[0 : 2])
			adjustBoxLimits(&obox, segment.Args[1 : 2])
		case sfnt.SegmentOpCubeTo:
			adjustBoxLimits(&cbox, segment.Args[0 : 3])
			adjustBoxLimits(&obox, segment.Args[2 : 3])
		default:
			panic("unexpected segment.Op")
		}
	}
	return cbox, obox
}

func adjustBoxLimits(box *fixed.Rectangle26_6, points []fixed.Point26_6) {
	for _, point := range points {
		if box.Max.X < point.X { box.Max.X = point.X }
		if box.Min.X > point.X { box.Min.X = point.X }
		if box.Max.Y < point.Y { box.Max.Y = point.Y }
		if box.Min.Y > point.Y { box.Min.Y = point.Y }
	}
}

// Computes how much the [control box] of the given segments exceeds
// the box defined by the "ON" contour points. Whenever there's an
// excess, that means that the control box doesn't match the bounding
// box of the glyph segments, which might have unintended effects in
// the rendering position of the glyph. Though you'd have to be crazy
// to care much about this, as the effect is almost always way smaller
// than typical hinting distortions. So, visually you are unlikely to
// see anything at all even if CBoxBadness are non-zero... but it has
// implications for technical correctness of computed left and right
// side bearings and stuff like that if you are obsessive enough.
//
// Returned badnesses are left, right, top and bottom, and the values
// can only be zero or positive.
//
// [control box]: https://freetype.org/freetype2/docs/glyphs/glyphs-6.html#section-2
func CBoxBadness(segments sfnt.Segments) (fixed.Int26_6, fixed.Int26_6, fixed.Int26_6, fixed.Int26_6) {
	cbox, obox := cboxObox(segments)
	leftBadness   := -cbox.Min.X + cbox.Min.X
	rightBadness  :=  cbox.Max.X - obox.Max.X
	topBadness    := -cbox.Min.Y + obox.Min.Y
	bottomBadness :=  cbox.Max.Y - obox.Max.Y
	return leftBadness, rightBadness, topBadness, bottomBadness
}

// Returns the ascent of the given rune both as units and as the ratio
// to the font's em square size. In general, capital latin latters will
// return ratios around 0.7, while lowercase letters like 'a', 'x', 'r'
// and similar will return ratios around 0.48. But anything is possible,
// really.
//
// The buffer can be nil.
func RuneAscent(font *sfnt.Font, codePoint rune, buffer *sfnt.Buffer) (sfnt.Units, float64, error) {
	if buffer == nil { buffer = &sfnt.Buffer{} }
	unitSize := fixed.Int26_6(font.UnitsPerEm())
	glyphIndex, err := font.GlyphIndex(buffer, codePoint)
	if err != nil { return 0, 0, err }
	contours, err := font.LoadGlyph(buffer, glyphIndex, unitSize, nil)
	if err != nil { return 0, 0, err }
	ascentUnits  := -contours.Bounds().Min.Y
	emProportion := float64(ascentUnits)/float64(unitSize)
	return sfnt.Units(ascentUnits), emProportion, nil
}

// TODO: a true BBox(segments sfnt.Segments) implementation
//       do double or triple pass. if cboxBadness is 0, we
//       already know the BBox. otherwise, do first a general
//       approximation with the normal segments and points, and
//       only at the end check if the bézier curves that may
//       affect the final bounding box actually affect it. or
//       just look into freetype implementation.
