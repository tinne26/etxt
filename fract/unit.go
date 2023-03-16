package fract

// Fixed point type to represent fractional values used for font rendering.
// 
// 26 bits represent the integer part of the value, while the remaining 6 bits
// represent the decimal part. For an intuitive understanding, if you can
// understand that var ms Millis = 1000 is storing the equivalent to 1 second,
// with Unit, instead of thousandths of a value, you are storing 64ths. So,
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

// Returns only the fractional part of the Unit.
// TODO: what about negative values?
func (self Unit) Fract() Unit {
	return self % 64
}

func (self Unit) Mul(multiplier Unit) Unit {
	mx64 := int64(self)*int64(multiplier)
	return Unit((mx64 + 32) >> 6)
}

func (self Unit) ToFloat64() float64 {
	return float64(self)/64.0 // *
	// math.Ldexp(float64(self), -6) also sounds good and works, but it's
	// slower. even with amd64 assembly, lack of inlining kills perf.
	// also, https://go-review.googlesource.com/c/go/+/291229
}

// TODO: method to decompose in floor + positive fract for glyph drawing?

// Defaults to [Unit.ToIntHalfUp](). For the fastest possible
// conversion to int, use [Unit.ToIntFloor]() instead.
func (self Unit) ToInt() int {
	return self.ToIntHalfUp()
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
