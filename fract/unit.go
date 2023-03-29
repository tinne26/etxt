package fract

// Fixed point type to represent fractional values used for font rendering.
// 
// 26 bits represent the integer part of the value, while the remaining 6 bits
// represent the decimal part. For an intuitive understanding, if you can
// understand that var ms Millis = 1000 is storing the equivalent to 1 second,
// with Unit, instead of 1/1000ths of a value, you are storing 1/64ths. So,
// var pixels Unit = 64 would mean 1 pixel, and 96 would be 1.5 pixels.
//
// The internal representation is compatible with [fixed.Int26_6].
//
// [fixed.Int26_6]: golang.org/x/image/math/fixed.Int26_6
type Unit int32

// Returns whether the Unit is a whole number or if it
// has a fractional part.
func (self Unit) IsWhole() bool {
	return self & 0x3F == 0
}

// Returns the absolute value of the unit.
func (self Unit) Abs() Unit {
	if self >= 0 { return self }
	return -self
}

// Returns only the fractional part of the unit.
func (self Unit) Fract() Unit {
	return self % 64
}

// Returns only the fractional part of the unit as a
// non-negative value relative to the unit's floor.
func (self Unit) FractShift() Unit {
	return self & 0x3F
}

// Multiplies the unit by the given value, rounding away from zero.
func (self Unit) Mul(multiplier Unit) Unit {
	mx64 := int64(self)*int64(multiplier)
	if mx64 >= 0 { return Unit((mx64 + 32) >> 6) }
	return Unit((mx64 + 31) >> 6)
}

// Note: I also tested this, but of course sometimes +1 results are
// closer due to truncation... and I just don't think there's any
// good use-case for it. Worsening precision to avoid one addition
// doesn't seem healthy.
// func (self Unit) MulTrunc(multiplier Unit) Unit {
// 	return Unit(int64(self)*int64(multiplier) >> 6)
// }

// Multiplies the unit by the given value, rounding up in case of ties.
func (self Unit) MulUp(multiplier Unit) Unit {
	mx64 := int64(self)*int64(multiplier)
	return Unit((mx64 + 32) >> 6) // round up
}

// Multiplies the unit by the given value, rounding down in case of ties.
func (self Unit) MulDown(multiplier Unit) Unit {
	mx64 := int64(self)*int64(multiplier)
	return Unit((mx64 + 31) >> 6) // round down
}

// Divides the unit by the given divisor, rounding away from zero in
// case of ties.
func (self Unit) Div(divisor Unit) Unit {
	// I don't know why people share obviously lame formulas for fixed
	// point division on the internet. Sure, they are fast and whatever...
	// but the results are so obviously off. So I decided to figure it out
	// on my own. The key idea is that we need a rounding factor to be 
	// applied before the actual division, unlike in the multiplication
	// where we apply the rounding afterwards. The natural rounding factor
	// here would be divisor/2, but if divisor is odd, this will result
	// in a slightly incorrect rounding value that will make the operation
	// fail in some cases. Instead, if we multiply everything by 2 again,
	// since we have the bits for it, there's no problem using 'divisor'
	// directly as the rounding factor. Well, there's also some sign
	// trickiness, but that's expanded below, you can figure it out.
	numerator   := int64(self)    << 7
	denominator := int64(divisor) << 1
	if (self >= 0) == (divisor >= 0) { // *
		numerator += int64(divisor)
	} else {
		numerator -= int64(divisor)
	}
	return Unit(numerator/denominator)
	// * If you wanted to round towards zero, instead, you would
	// have to expand the (self >= 0) == (divisor >= 0) expression
	// into something like this:
	//    if self >= 0 {
	// 	   if divisor >= 0 { // +/+
	// 		   numerator += int64(divisor) - 1
	// 	   } else { // +/-
	// 		   numerator -= int64(divisor) + 1
	// 	   }
	//    } else { // self < 0
	// 	   if divisor >= 0 { // -/+
	// 		   numerator -= int64(divisor) - 1
	// 	   } else { // -/-
	// 		   numerator += int64(divisor) + 1
	// 	   }
	//    }
	// You can try yourself: changing the +/- values you can make
	// it adjust the rounding. I only picked the simplest version.
	// The tests include a debugRounding variable that can be used
	// to visualize the results of any roundings you may want to
	// experiment with.
}

// Rescales the value from the 'from' scale to the 'to' scale,
// rounding away from zero. In etxt, this is often used to rescale font
// metrics between different EM sizes (e.g. an advance of 512 on a
// font with EM of 1024 units corresponds to an advance of 384 with
// an EM size of 768, or 512.Rescale(1024, 768) = 384).
func (self Unit) Rescale(from, to Unit) Unit {
	// this is basically an inlined form of self.Mul(to).Div(from)
	// that avoids rounding between operations. refer to them for
	// further implementation details
	numerator   := (int64(self)*int64(to)) << 1
	denominator := int64(from) << 1
	if (numerator >= 0) == (from >= 0) {
		numerator += int64(from)
	} else {
		numerator -= int64(from)
	}
	return Unit(numerator/denominator)
}

// The conversion is always exact.
func (self Unit) ToFloat64() float64 {
	return float64(self)/64.0 // *
	// math.Ldexp(float64(self), -6) also sounds good and works, but it's
	// slower. even with amd64 assembly, lack of inlining kills perf.
	// also, https://go-review.googlesource.com/c/go/+/291229
}

// The conversion is exact in the +/-16777216 Units range. Beyond that
// range, which corresponds to +/-2^18 (+/-262144) in the decimal
// numbering system), conversions become progressively less precise.
func (self Unit) ToFloat32() float32 {
	return float32(self)/64.0
}

// Defaults to [Unit.ToIntHalfAway](0). For the fastest possible
// conversion to int, use [Unit.ToIntFloor]() instead.
func (self Unit) ToInt() int {
	return self.ToIntHalfAway(0)
}

// Fastest conversion from Unit to int.
func (self Unit) ToIntFloor() int {
	return (int(self) +  0) >> 6
}

func (self Unit) ToIntCeil() int {
	return (int(self) + 63) >> 6
}

func (self Unit) ToIntToward(reference int) int {
	floor := self.ToIntFloor()
	if floor >= reference { return floor }
	return self.ToIntCeil()
}

func (self Unit) ToIntAway(reference int) int {
	ceil := self.ToIntCeil()
	if ceil > reference { return ceil }
	return self.ToIntFloor()
}

func (self Unit) ToIntHalfDown() int {
	return (int(self) + 31) >> 6
}

func (self Unit) ToIntHalfUp() int {
	return (int(self) + 32) >> 6
}

func (self Unit) ToIntHalfToward(reference int) int {
	if self >= FromInt(reference) { return self.ToIntHalfDown() }
	return self.ToIntHalfUp()
}

func (self Unit) ToIntHalfAway(reference int) int {
	if self <= FromInt(reference) { return self.ToIntHalfDown() }
	return self.ToIntHalfUp()
}

func (self Unit) Floor() Unit {
	return self & ^0x3F
}

func (self Unit) Ceil() Unit {
	return (self + 0x3F).Floor()
}

func (self Unit) Toward(reference int) Unit {
	if self >= FromInt(reference) { return self.Floor() }
	return self.Ceil()
}

func (self Unit) Away(reference int) Unit {
	if self <= FromInt(reference) { return self.Floor() }
	return self.Ceil()
}

func (self Unit) HalfDown() Unit {
	return (self + 31).Floor()
}

func (self Unit) HalfUp() Unit {
	return (self + 32).Floor()
}

func (self Unit) HalfToward(reference int) Unit {
	if self >= FromInt(reference) { return self.HalfDown() }
	return self.HalfUp()
}

func (self Unit) HalfAway(reference int) Unit {
	if self <= FromInt(reference) { return self.HalfDown() }
	return self.HalfUp()
}

// Given a fractional step between 1 and 64, it quantizes the
// Unit to that fractional value, rounding up in case of ties.
func (self Unit) QuantizeUp(step Unit) Unit {
	// safety assertions
	if step > 64 { panic("step > 64") }	
	if step <  1 { panic("step < 1" ) }

	// quantize based on the fraction relative to floor
	lfract := self & 0x3F
	mod    := lfract % step
	if mod == 0 { return self }
	sum := lfract - mod
	if mod >= ((step + 1) >> 1) { // tie point
		sum += step
		if sum > 64 { sum = 64 }
	}
	return self.Floor() + sum
}

// Given a fractional step between 1 and 64, it quantizes the
// Unit to that fractional value, rounding down in case of ties.
func (self Unit) QuantizeDown(step Unit) Unit {
	// safety assertions
	if step > 64 { panic("step > 64") }	
	if step <  1 { panic("step < 1" ) }

	// quantize based on the fraction relative to floor
	lfract := self & 0x3F
	mod    := lfract % step
	if mod == 0 { return self }
	sum := lfract - mod
	if mod > (step >> 1) { // tie point
		sum += step
		if sum > 64 { sum = 64 }
	}
	return self.Floor() + sum
}
