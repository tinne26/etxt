// CPU-based vectorial text rendering has always been a performance
// sensitive task, with glyph rasterization being one of the most
// critical steps in the process. One of the techniques that can be
// used to improve the rasterization speed is using [fixed point]
// arithmetic instead of floating point arithmetic — and that's what
// brings us to this subpackage.
//
// The fract subpackage defines a [Unit] type representing a 26.6
// fixed point value and provides numerous methods to perform fixed
// point operations. Additionally, the subpackage also defines the
// [Point] and [Rect] helper types.
//
// Other font related Golang packages tend to depend on
// [golang.org/x/image/math/fixed] instead, but this subpackage
// offers more methods, more accurate algorithms and is designed
// to integrate directly with etxt.
//
// [fixed point]: https://github.com/tinne26/etxt/blob/v0.0.9/docs/fixed-26-6.md
package fract
