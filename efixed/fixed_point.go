// efixed is a utility subpackage containing functions for working with
// fixed point [fixed.Int26_6 numbers]. You most likely will never need
// to use this package, but if you are rolling your own emask.Rasterizer
// or ecache.GlyphCacheHandler maybe you find something useful here.
//
// [fixed.Int26_6 numbers]: https://github.com/tinne26/etxt/blob/main/docs/fixed-26-6.md
package efixed

import "math"
import "strconv"
import "golang.org/x/image/math/fixed"

// Converts a value from its fixed.Int26_6 representation to its float64
// representation. Conversion is always exact.
func ToFloat64(value fixed.Int26_6) float64 {
	return float64(value)/64.0
}

// Converts the given float64 to the nearest fixed.Int26_6.
// If there's a tie, returned values will be different, and
// the first will always be smaller than the second.
//
// The function will panic if the given float64 is not closely 
// representable by any fixed.Int26_6 (including Inf, -Inf and NaN).
func FromFloat64(value float64) (fixed.Int26_6, fixed.Int26_6) {
	// TODO: overflows may still be possible, and faster conversion
	//       methods must exist, but go figure
	candidateA := fixed.Int26_6(value*64)
	diffA := abs64(float64(candidateA)/64.0 - value)
	if diffA == 0 { return candidateA, candidateA } // fast exact conversion

	// fast path didn't succeed, proceed now to the more complex cases

	// check NaN
	if math.IsNaN(value) {
		panic("can't convert NaN to fixed.Int26_6")
	}

	// check bounds
	if value > 33554431.984375 {
		if value <= 33554432 {
			result := fixed.Int26_6(0x7FFFFFFF)
			return result, result
		}
		given := strconv.FormatFloat(value, 'f', -1, 64)
		panic("can't convert " + given + " to fixed.Int26_6, the biggest representable value is 33554431.984375")
	} else if value < -33554432 {
		if value >= -33554432.015625 {
			result := -fixed.Int26_6(0x7FFFFFFF) - 1
			return result, result
		}
		given := strconv.FormatFloat(value, 'f', -1, 64)
		panic("can't convert " + given + " to fixed.Int26_6, the smallest representable value is -33554432.0")
	}

	// compare current candidate with the next and previous ones
	candidateB := candidateA + 1
	candidateC := candidateA - 1
	diffB := abs64(float64(candidateB)/64.0 - value)
	diffC := abs64(float64(candidateC)/64.0 - value)

	if diffA < diffB {
		if diffA == diffC { return candidateC, candidateA }
		if diffA  < diffC { return candidateA, candidateA }
		return candidateC, candidateC
	} else if diffB < diffA {
		if diffB == diffC { panic(value) } // this shouldn't be possible, but just to be safe
		if diffB  < diffC { return candidateB, candidateB }
		return candidateC, candidateC
	} else { // diffA == diffB
		return candidateA, candidateB
	}
}

// Same as [FromFloat64](), but returning a single value.
// In case of ties, the result closest to zero is selected.
func FromFloat64RoundToZero(value float64) fixed.Int26_6 {
	a, b := FromFloat64(value)
	if a >= 0 { return a } // both values are positive, a is smallest one
	return b // both values are negative, b is the closest to zero
}

// Same as [FromFloat64](), but returning a single value.
// In case of ties, the result furthest away from zero is selected.
func FromFloat64RoundAwayZero(value float64) fixed.Int26_6 {
	a, b := FromFloat64(value)
	if a >= 0 { return b } // both values are positive, b is the biggest one
	return a // both values are negative, a is the smallest one
}

// Handy method to convert int values to their exact fixed.Int26_6
// representation. [fixed.I]() also does this, but this has bound checks,
// in case that's important for you.
//
// [fixed.I]: https://pkg.go.dev/golang.org/x/image/math/fixed#I
func FromInt(value int) fixed.Int26_6 {
	// bound checks
	if value > 33554431 {
		given := strconv.Itoa(value)
		panic("can't convert " + given + " to fixed.Int26_6, the biggest representable int is 33554431")
	} else if value < -33554432 {
		given := strconv.Itoa(value)
		panic("can't convert " + given + " to fixed.Int26_6, the smallest representable int is -33554432")
	}

	// actual conversion
	return fixed.Int26_6(value << 6)
}

// Notice: the following methods can overflow, but it's by such a small
//         margin with regards to actual overflows that it's not even being
//         mentioned in the documentation. Usage of this package for etxt
//         shouldn't get even closer to values that can overflow, and in
//         fact etxt caches will impose lower limits on fixed.Int26_6
//         magnitudes on their own already.

// Like [fixed.Floor](), but returning the fixed.Int26_6 value instead
// of an int.
//
// [fixed.Floor]: https://pkg.go.dev/golang.org/x/image/math/fixed#Int26_6.Floor
func Floor(value fixed.Int26_6) fixed.Int26_6 {
	return (value & ^0x3F)
}

// Like [fixed.Round](), but returns the fixed.Int26_6 instead of an int
// and is clearly named. For the int result, see [ToIntHalfUp]() instead.
//
// [fixed.Round]: https://pkg.go.dev/golang.org/x/image/math/fixed#Int26_6.Round
func RoundHalfUp(value fixed.Int26_6) fixed.Int26_6 {
	return (value + 32) & ^0x3F
}

// Like [RoundHalfUp](), but rounding down. For the int result, see
// [ToIntHalfDown]() instead.
func RoundHalfDown(value fixed.Int26_6) fixed.Int26_6 {
	return (value + 31) & ^0x3F
}

// Like [RoundHalfUp](), but rounding away from zero. For the int result, see 
// [ToIntHalfAwayZero]() instead.
func RoundHalfAwayZero(value fixed.Int26_6) fixed.Int26_6 {
	if value >= 0 { return RoundHalfUp(value) }
	return RoundHalfDown(value)
}

// Like [RoundHalfUp](), but directly converting to int.
func ToIntHalfUp(value fixed.Int26_6) int { return int(value + 32) >> 6 }

// Like [RoundHalfDown](), but directly converting to int.
func ToIntHalfDown(value fixed.Int26_6) int { return int(value + 31) >> 6 }

// Like [RoundHalfAwayZero](), but directly converting to int.
func ToIntHalfAwayZero(value fixed.Int26_6) int {
	if value >= 0 { return ToIntHalfUp(value) }
	return ToIntHalfDown(value)
}

// Doesn't care about NaNs and general floating point quirkiness.
func abs64(value float64) float64 {
	if value >= 0 { return value }
	return -value
}
