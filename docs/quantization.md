# Quantization

Quantization in the context of **etxt** refers to the process of adjusting glyph coordinates to the pixel grid.

Whenever a glyph is drawn, we need to update the drawing position for the next glyph. Quite often, the new position won't fall perfectly at the start of a pixel, but rather at a fractional pixel position. What should we do then? Jump directly to the next whole pixel? Try to draw the glyph at this fractional position?

Before deciding, we need to explain the trade-offs:
- If we try to draw glyphs at their exact fractional positions, we will be respecting the flow of the text as much as possible... but we will potentially have to cache each glyph in a variety of fractional positions. The maximum fractional precision per axis is 1/64th of a pixel. This means that if we don't quantize text at all in the horizontal axis, we may have to store each glyph up to 64 times in our cache, all with very slightly different positions. If we are also not quantizing in the vertical axis, the possibilities are multiplied by 64 again, resulting in a maximum of 4096 sub-positions per glyph. That's... not ideal.
- If we fully quantize glyph positions to the pixel grid, we only need to store "one variation" of each glyph, which is great, but the flow of the text may be slightly off.

## So what do we do? Quantize or not quantize?

By default, **etxt** does full glyph quantization (aligns glyphs to the pixel grid), but you can modify this behavior through `Renderer.SetQuantizerStep()`. The best choice, though, will depend on the situation:
- When you are working with big text (e.g. >=32px), you rarely want to consider fractional pixel positions. Leave quantization on, let the renderer align glyphs to the pixel grid and call it a day, no one will be the wiser.
- When you are working with small text, you generally want to increase the precision of the text's horizontal positioning if you want high quality results. Not everyone can tell, and the font being used and other variables can make this more or less necessary, but I can tell and it's more pleasant to have *some respect* for the fractional positions. Setting `step = 22` (1/3rd of a pixel) or `step = 16` (1/4th of a pixel) with `Renderer.SetQuantizerStep(16, 64)` is virtually always enough.
- You almost always want to keep the vertical positions quantized to the pixel grid. Text flows horizontally, not vertically, so there's no need to be precise in the vertical axis (it will waste space in the glyphs cache and it can even look worse in many cases).
- The main exception to the previous rules is when you want to animate text to give it movement. If you start moving your text but keep it quantized to the pixel grid, the movement will look jittery and jumpy. So, if you need high quality text movement animations, specially when they are very slow, you will have to respect the fractional positions, even if you are working with big text. I'd use at least `step = 16` on animations that aren't too slow.
- Another exception is if you aren't using a cache. I don't know when or why would you do that, but if you are not using a cache you may as well go with the maximum precision (`Renderer.SetQuantizerStep(1, 1)`), because you are fully recomputing glyph masks each time anyway.

## Visual comparison

Here's an example from [examples/gtxt/quantization](https://github.com/tinne26/etxt/blob/v0.0.9-alpha.7/examples/gtxt/quantization/main.go):
![](https://github.com/tinne26/etxt/blob/v0.0.9-alpha.7/docs/img/gtxt_quantization.png?raw=true)

The differences are visible if you start comparing the lines letter by letter, but they are also not major enough to be obvious if you aren't focusing on them. Different sizes and fonts may produce different results. For example, monospaced fonts and fonts in a pixelated style may look more consistent with quantization, while fonts with more natural or hand drawn styles will almost always flow better without full horizontal quantization.
