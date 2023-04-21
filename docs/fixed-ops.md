# Fixed point multiplication and division

When working with fonts, we need to use fixed point values with 26 bits for the integer part and 6 bits for the decimal part. Then you may want to figure out how to multiply and divide, search on the internet and... get incorrect formulas.


## Multiplication

So, let's say we have 2000ms and 2500ms. In other words, 2s and 2.5s. If we multiply them, we should get 2*2.5 = 5s, which converted back to milliseconds, is 5000ms. Cool.

If you actually multiply 2000*2500, you will see you are off by a factor of 1k. So, we would need to divide by that after multiplying.

So we go and apply these ideas, and the first trivial formula that seems to work for multiplying fixed 26.6 integers is the following:
```Golang
func (a Fixed26_6) Mul(b Fixed26_6) Fixed26_6 {
	return (a*b)/64
}
```
Here we don't have a factor of 1k, but rather 64, as that's 2^6 = 64, the number of values representable in the decimal part of a `Fixed26_6` number. With milliseconds, the number of values representable in the decimal part was 1000 (from 0 to 999) instead.

This is cool, but this formula is not correct yet. Which is annoying, because in many places you will see this formula used as if it was the correct way to multiply fixed integers. Staying with the milliseconds parallel, if you look a bit deeper and try with a few more values, you see that if you multiply 1005 by 2503, you end up with 2515515. If we divide this value by 1000 with an integer division, we lose the 515 part, that would be the unrepresentable decimal part. But this part should make us jump from 2515 to 2516! So, we need to take those digits into account to *round the result*.

Rounding the result is simple enough: before dividing, we need to add a number that will make the digits that we can represent go up if necessary. For example, when we are dividing by 1000, since the first number we would have to round up is 500, we use that value. So, when we get 2515515, we add 500 to it, and we now have 2516015. When we discard the lower digits by dividing by 1000, our result is already rounded and we get the correct 2516 result.

You can also round down by using 499 intead of 500. You can also take sign into account to round away from zero or other ideas. Now multiplication would work. Going back to our `Fixed26_6` context:
```Golang
func (a Fixed26_6) Mul(b Fixed26_6) Fixed26_6 {
	return (a*b + 32)/64
}
```
Optimizing a bit more and taking overflows into account, we can improve the situation with the following code:
```Golang
func (a Fixed26_6) Mul(b Fixed26_6) Fixed26_6 {
	return Fixed26_6((int64(a)*int64(b) + 32) >> 6)
}
```


## Division

Division, a priori, doesn't look that different from multiplication. Say we have 5000ms and want to divide by 2000ms. 5s/2s = 2.5s. If we divide 5000/2000, in integer arithmetic, we will get 2. Oops, we are missing the decimals and the scaling as milliseconds, so... what if we multiply by 1000 *before* the division?
```Golang
func (a Fixed26_6) Div(b Fixed26_6) Fixed26_6 {
	return a*64/b
}
```

Now it works. With milliseconds instead of `Fixed26_6`, we would get 5000*1000/2000 = 2500, in integer arithmetic! Problem solved... or not?

As with multiplication, if we use values with more relevant digits in the decimal part, we start getting in trouble. Say 5005/2003. The result of this operation, in decimal arithmetic, is 2.498751... In terms of milliseconds, we should be getting 2499 as the most accurate result. But again, if we simply rely on integer arithmetic, this is not going to happen, and the remaining decimals will be clamped... So, what do we do? Can we add 500 like the previous time? Well, in this case it does actually work, but in the general case, it doesn't. The reason is that the divisor this time is not 1000, but some arbitrary value. We can halve that value, also taking into account if it's even or odd, but... even then it won't be enough. Let's see what we have:

```Golang
func (a Fixed26_6) Div(b Fixed26_6) Fixed26_6 {
	return Fixed26_6((int64(a)*64 + int64(b)/2)/int64(b))
}
```
There are a few problems here:
- Signs matter. If `a` and `b` signs are different, we won't be properly adding a rounding factor as intended, but rather bringing the numerator further away from what it should be.
- Assume `a` and `b` positive. If `b` is even, then `b/2` will round up, but if `b` is odd, `b/2` will round down. You may think this can be solved by adding 1 to `b` before dividing it, but then you are misrepresenting the actual value and in some cases the operation fails. Yikes.

There are multiple ways to solve this. Signs are not a big problem, it's only a tricky issue you need to be careful with and test appropriately. Change a sign here or there if necessary, check if numerator and denominator signs are the same, that stuff.

Rounding is the bigger problem. In many cases, one solution you can use is simply multiplying everything by 2 again, so the divisor forcefully becomes even. In `Fixed26_6`, for example, you have enough space to multiply by 128 (left shift by 7) instead of 64 (left shift by 6) without any danger. Alternatively, you can just hope to get a "close enough" result and then manually check the next and previous possible results, and see if any of them are better. There are some relatively not-super-slow ways to check for this.

Actual code is available on the `fract/` subpackage.
