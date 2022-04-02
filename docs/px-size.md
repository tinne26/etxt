# Font pixel sizes
So, the question that everyone is asking...

If you set the font size to 16px, how big will your text really be?

Well, while every font can do whatever it wants, in practice there's some general consensus:
- In most sane fonts, the height of a *capital* latin letter will typically be between 65% and 75% of the pixel size. So, if you are drawing with a font at 16px, an "A" will usually be between 10 and 12px tall.
- In most sane fonts, the x-height of a *lowercase* latin letter will typically be between 46% and 52% of the pixel size (the x-height is the height of lowercase letters without accounting for the ascenders and descenders of characters like "g", "p", "y" and "t"). So, if you are drawing with a font at 16px, characters like "x", "a", "r", "v" and similar will usually be between 7 and 9px tall.

These properties can actually be obtained from the [font metrics](https://pkg.go.dev/golang.org/x/image/font#Metrics) (see `CapHeight` and `XHeight`). Although not common, notice that they can still be wrong and not match the actual font. For example, [Carter One](https://fonts.google.com/specimen/Carter+One) reports an x-height of 473 when it's actually something like 1120.

When we talk about font size in pixels, what we are really defining is the [em](https://en.wikipedia.org/wiki/Em_(typography)) size. For example, a "j" —which has features going both upwards and downwards in its glyph— tends to be close to an em in vertical length. If you draw a "j" at 16px with **etxt** and measure its height, you will see it's 14px for most fonts (~add an empty pixel of padding up and down).

Display fonts designed for titles, hand-drawn fonts, pixelated fonts and similar are the most likely to break the general sizing rules, often with legit reasons to do so. Also, random fonts from the internet might do all kinds of weird stuff that I don't even want to start talking about.

Finally, remember that font sizes will also affect line height (unless you are controlling that manually with `SetLineHeight`).

## Taming sizes on the wild west
If you are working with a fixed set of fonts, you can usually try different sizes until you get the fonts to look more or less consistent and call it a day.

Sadly, if you don't know what fonts may be used (maybe you depend on system fonts or you let users specify their own), figuring out font sizes can become a real problem.

There are three things you can do (each one improving slightly on the previous):
- Set your renderer sizes expecting font sizes to be sane.
- Compute the size of a font dynamically based on the font's x-height and your own target x-height. In general I find x-height to be more reliable than cap-height, but you could also use both. When you do this, though, you should also set the renderer's line height manually.
- Allow users to set their own scaling factor for each font so they can get it to be balanced with the rest of the fonts. This generally only makes sense if you are letting users load their own fonts to define custom UI styles, though, which is quite uncommon.

## Why pixels instead of points?
You might have noticed that most software uses points (pt) for font sizes, instead of pixels. Why? I don't know. One reason might be that in the case of screens with non-square pixels, using points allows the software to adjust to different horizontal and vertical resolutions and make text look as it should. But tradition might also play a role.

By default, **etxt** doesn't bother with all that because:
- All non-ancient computer screens have square pixels. Some TVs have non-square pixels, but this is also becoming increasingly rare as SMPTE standards define the PAR (pixel aspect-ratio) to be 1 (square) for HD screens.
- DPIs are confusing as hell for most developers, specially game developers that would like to only have to care about pixels. Font sizes are already confusing enough on its own. And don't even get me started on DPI vs PPI...
- ~~Why are you people still using inches? Can't we ban DPI from the digital realm already?~~
- Golang alternatives like the **opentype** package (used by **ebiten/text**) ask for DPIs... but the same value is used horizontally and vertically (there's no real distinction in the code). This means that even if you wanted to work with non-square pixels and cared about DPIs/PPIs, the **opentype** DPI configuration wouldn't help you either.
- If you really want to mess with all this, you can still go and implement a custom rasterizer and a sizer in **etxt**.

I don't know why, but if you still want some more references on points vs pixels:
- When using Golang's **opentype** package, setting a DPI of 72 will make the **etxt** and **opentype** sizes match (at 72DPI, 1px = 1pt).
- Other programs often assume a DPI of 96. For example, CSS3 defines [`1px = 1/96th of 1 in`](https://www.w3.org/TR/css3-values/#absolute-lengths). So, if you wanted convert between **etxt** and css sizes, you would have to multiply by 4/3 (or by 3/4, depending on which direction you are going).
