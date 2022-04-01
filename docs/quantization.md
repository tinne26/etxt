# Quantization
Quantization in the context of **etxt** refers to the process of adjusting glyph coordinates to the pixel grid. If you don't quantize anything, each glyph may be drawn at a different fractional pixel position. This means that the flow of text will be more faithful to the original font scalable outlines, but it also means that the same character may be rendered in different ways based on its position.

Here's a quick summary of the modes available in **etxt**:
- No quantization: very cache unfriendly (up to 64*64 = 4096 positions per glyph), but technically the most precise option.
- Vertical-only quantization: respects the natural flow of horizontal text at a manageable cost. Rarely worth it with big text though.
- Full quantization: computationally the cheapest option, while still being reasonable to use in many situations.

Here's the example from [examples/gtxt/quantization](https://github.com/tinne26/etxt/examples/gtxt/quantization):
![](https://github.com/tinne26/etxt/docs/img/etxt_quantization.png)

The differences are visible if you start comparing the lines letter by letter, but they are also not major enough to be obvious if you aren't focusing on them. Different sizes and fonts may produce different results. For example, monospaced fonts and fonts in a pixelated style may look more consistent with quantization, while fonts with more natural or hand drawn styles may flow better without horizontal quantization.
