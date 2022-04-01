# Fixed.Int26_6
When working with fonts we often have to deal with **fixed precision** numbers.

You are probably already familiar with *floating* precision numbers (`float32`, `float64`), so fixed precision numbers should be fairly easier to understand: instead of having a mantissa and an exponent, we simply have a certain amount of bits reserved for the **whole part** of the number, and the remaining bits being used for the **decimal part**.

For example, with a fixed precision `int` that has 26 bits for the whole number and 6 bits for the decimal part, we can represent integers between `2^25 - 1 = 33554431` and `-2^25 = -33554432`, with up to `2^6 = 64` different decimal values for each whole number. The representable decimal magnitudes are all multiples of `1/64 = 0.015625`. We can store an `int26.6` in an `int32`.

To make it easier to interpret:
- If you had an `int32` representing milliseconds and wanted to know how many seconds you have, you would automatically know to divide by 1000. You know your `int32` represents thousandths of seconds, or 1/1000th parts of a second.
- If you have a fixed point `int26.6`, your `int32` represents 1/64th parts (remember that 64 comes from 2^6) of whatever you are measuring. Pixels in our case.

## So, why do we have to work with fixed precision numbers?
I don't know and I didn't bother to figure out, but the thing is that since fonts are scalable, sometimes we need to work with coordinates that do not exactly match the pixel grid, and fixed point numbers have been traditionally chosen to take care of this. Both 16.16 and 26.6 fixed point types are common when working with fonts, but only 26.6 is used with **etxt**.

The type used in Golang for fixed 26.6 values is defined in https://pkg.go.dev/golang.org/x/image/math/fixed. **etxt** also has a small [efixed](https://pkg.go.dev/github.com/tinne26/etxt/efixed) package which includes some useful helper functions.

## In which situations do we need fixed precision numbers?
- After drawing a glyph, the amount of space we need to advance to prepare for drawing the next glyph may leave us at a fractional pixel coordinate.
- From the previous point, if you are not quantizing fractional coordinates, you may have to start drawing text at a fractional pixel position. That's why `Draw*Fract*()` functions exist and why `Traverse` functions use `fixed.Int26_6` values.
- Glyph rasterizers need to be able to deal with fractional pixel positions.

## Practical advice for operating with fixed precision numbers
Most of the time operating with fixed precision numbers is quite easy, and you only need to do one of the following:
- Use the right rounding function (see [efixed](https://pkg.go.dev/github.com/tinne26/etxt/efixed) helper package).
- Convert from/to integer coordinates by shifting by 6 (multiplying or dividing by 64).
- Convert from/to actual `float` coordinates (also by casting and multiplying or dividing by 64).

Quick sample snipet:
```Golang
// convert from int to fixed26.6 by multiplying by 64
myInt := 100
fixedValue := fixed.Int26_6(myInt << 6)

// add 0.5 to the fixed value
fixedValue += 32 // 64 would add "1", so 32 is half that, 0.5

// convert to float64 and display
rawFloatValue  := float64(fixedValue)
normFloatValue := rawFloatValue/64.0 // remember the example of milliseconds!!
fmt.Printf("value = %f\n", normFloatValue) // prints "value = 100.50000"
```

You rarely need to do complex operations with fixed point values, so... focus on round, truncate, floor, and multiplying and dividing by 64 to convert between representations, assisted by bit shifts where possible. 95% of the time that's all you need.
