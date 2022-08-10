# Fixed.Int26_6
When working with fonts we often have to deal with **fixed precision** numbers.

You are probably already familiar with *floating* precision numbers (`float32`, `float64`), so fixed precision numbers should be fairly easier to understand: instead of having a mantissa and an exponent, we simply have a certain amount of bits reserved for the **whole part** of the number, and the remaining bits being used for the **decimal part**.

For example, with a fixed precision `int` that has 26 bits for the whole number and 6 bits for the decimal part, we can represent integers between `2^25 - 1 = 33554431` and `-2^25 = -33554432`, with up to `2^6 = 64` different decimal values for each whole number. The representable decimal magnitudes are all multiples of `1/64 = 0.015625`. We can store an `int26.6` in an `int32`.

To make it easier to interpret:
- If you had an `int32` representing milliseconds and wanted to know how many seconds you have, you would automatically know to divide by 1000. You know your `int32` represents thousandths of seconds, or 1/1000th parts of a second.
- If you have a fixed point `int26.6`, your `int32` represents 1/64th parts (remember that 64 comes from 2^6) of whatever you are measuring. Pixels in our case.

## So, why do we have to work with fixed precision numbers?
I don't know and I didn't bother to figure out, but the thing is that since font outlines are scalable, sometimes we need to work with coordinates that do not exactly match the pixel grid, and fixed point numbers have been traditionally chosen to take care of this. Both 16.16 and 26.6 fixed point types are common when working with fonts, but only 26.6 is used within **etxt**.

## In which situations do we need fixed precision numbers?
- After drawing a glyph, the amount of space we need to advance to prepare for drawing the next glyph may leave us at a fractional pixel coordinate.
- From the previous point, if you are not quantizing fractional coordinates, you may have to start drawing text at a fractional pixel position. That's why `DrawFract()` exists and why `Traverse*` functions use `fixed.Int26_6` values.
- Glyph rasterizers need to be able to deal with fractional pixel positions.

## Practical advice for operating with fixed precision numbers
There are two key packages to be aware of when dealing with fixed precision numbers:
- The Golang package where they are defined, [x/image/math/fixed](https://pkg.go.dev/golang.org/x/image/math/fixed).
- The [etxt/efixed](https://pkg.go.dev/github.com/tinne26/etxt/efixed) subpackage, which contains a few additional helpful functions.

Most of the time, to operate with fixed precision numbers you only need to do one of the following:
- Use the right rounding function, like [`Ceil()`](https://pkg.go.dev/golang.org/x/image/math/fixed#Int26_6.Ceil), [`Floor()`](https://pkg.go.dev/golang.org/x/image/math/fixed#Int26_6.Floor) and [`efixed.ToIntHalfUp()`](https://pkg.go.dev/github.com/tinne26/etxt/efixed#ToIntHalfUp) and its variants.
- Convert from/to integer coordinates:
	- To convert from `int` to `fixed.Int26_6` you can use [`efixed.FromInt()`](https://pkg.go.dev/github.com/tinne26/etxt/efixed#FromInt).
	- To convert from `fixed.Int26_6` to `int`, you generally round  the `fixed.Int26_6` variable itself with [`Ceil()`](https://pkg.go.dev/golang.org/x/image/math/fixed#Int26_6.Ceil).
- Convert from/to actual `float64` coordinates:
	- To convert from `float64` to `fixed.Int26_6` you use [`efixed.FromFloat64()`](https://pkg.go.dev/github.com/tinne26/etxt/efixed#FromFloat64) and its variants.
	- To convert from `fixed.Int26_6` to `float64` you use [`efixed.ToFloat64()`](https://pkg.go.dev/github.com/tinne26/etxt/efixed#ToFloat64).

Quick sample snipet:
```Golang
// convert from int to fixed26.6
myInt := 100
fixedValue := efixed.FromInt(myInt) // == fixed.Int26_6(myInt << 6)

// add 0.5 to the fixed value
fixedValue += 32 // 64 would add "1", so 32 is half that, 0.5

// convert to float64 and display
floatValue  := efixed.ToFloat64(fixedValue) // == float64(fixedValue)/64.0
fmt.Printf("value = %f\n", floatValue) // prints "value = 100.50000"
```

