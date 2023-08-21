# Tips for pixel-art-like vectorial fonts

Pixel-art-like fonts can be used with etxt as long as they also contain glyph outlines (as opposed to fonts with only glyph bitmaps).

Quick practical advice:
- Enable full horizontal quantization with [`Renderer.Complex().SetHorzQuantization(etxt.QtFull)`](https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.6#RendererFract.SetHorzQuantization).
- Pixel-art-like vectorial fonts are designed for a single size (or their multiples). 16px is common... but single sizes have some implications:
	- You don't want to use [Renderer.SetScale](). Leave the scale to 1. You want to be rendering on your logical, fixed-size canvas, and only doing scaling later on the projection from the logical canvas to the full resolution screen.
	- If your font doesn't look sharp at the intended size, DPI may be to blame. There are two common DPI values used in the wild: 72DPI and 96DPI. On etxt, 72DPI is implicitly used. If a font is designed for 96DPI, you may need to multiply its size by 4/3 (TODO: or was it 3/4? test and confirm) and use [`Renderer.Fract().SetSize()`](https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9-alpha.6#RendererFract.SetSize) to set the fractional size if necessary.

In general, etxt is not optimized or oriented to pixel art fonts, and sfnt, the underlying library used to parse the fonts doesn't have support for glyph bitmaps. This doesn't mean that using etxt is crazy if you are working with such fonts; etxt still provides many useful features no matter the type of font you are using. That being said, if a specialized package existed for dealing with this kind of fonts on Ebitengine, that could easily become a better alternative. Sadly, I'm not aware of any such package existing (the best attempt at the moment seems to be [ingenten](https://github.com/Frabjous-Studios/ingenten)).
